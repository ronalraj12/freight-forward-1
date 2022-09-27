package handlers

import (
	"fmt"
	"github.com/RemoteState/yourdaily-server/dbHelpers"
	"github.com/RemoteState/yourdaily-server/firebase"
	"github.com/RemoteState/yourdaily-server/middlewares"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

func FindAndPing(mode models.OrderMode, addressID, userID, orderID int) {
	UserLocation, err := dbHelpers.SelectAddressWithID(userID, addressID, true)
	if err != nil {
		logrus.Error(err)
		return
	}
	allStaff, err := dbHelpers.SelectLocationOfStaff(mode)
	if err != nil {
		logrus.Error(err)
		return

	}
	staffFound := make([]int64, 0)

	for _, v := range allStaff {
		dist := utils.GeoDistance(v.Long.Float64, v.Lat.Float64, UserLocation.Long, UserLocation.Lat)
		if dist < float64(models.RadiusForSearch) {
			staffFound = append(staffFound, int64(v.StaffID.Int))
		}
	}
	if len(staffFound) == 0 {
		logrus.Infof("number of users : %d userIDs %v", len(staffFound), staffFound)
		return
	}
	logrus.Printf("sending the notification to %v", staffFound)

	err = firebase.SendNewOrderNotificationToStaff(staffFound, orderID, UserLocation.Lat, UserLocation.Long, UserLocation.AddressData)
	if err != nil {
		logrus.Errorf("SendNewOrderNotificationToStaff: Error while sending push notifications %v", err)
	}

}

//OrderNow POST /api/user//order/now
func OrderNow(w http.ResponseWriter, r *http.Request) {
	userID := middlewares.UserContext(r).ID

	newOrder := models.Order{
		UserID:       userID,
		DeliveryTime: time.Now().Format(time.RFC3339Nano),
	}

	if err := utils.ParseBody(r.Body, &newOrder); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}
	if len(newOrder.Items) == 0 {
		newOrder.Mode = models.CartMode
	} else {
		newOrder.Mode = models.DeliveryMode
	}
	flagCount, _, err := dbHelpers.GetFlagCountAndLastOrderStatus(userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	if flagCount >= models.MaxFlagCount {
		err = fmt.Errorf("Account Blocked")
		if err != nil {
			utils.RespondError(w, http.StatusNotAcceptable, err, err.Error(), err.Error())
			return
		}
	}

	addressLocation, err := dbHelpers.SelectAddressWithID(newOrder.UserID, newOrder.AddressID, false)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid address id")
		return
	}
	smId, err := dbHelpers.StoreManagerNearMe(addressLocation.Lat, addressLocation.Long)
	if err != nil {
		utils.RespondError(w, http.StatusNotAcceptable, err, err.Error(), err.Error())
		return
	}
	newOrder.StoreMangerID = smId

	newOrder.Items = utils.FilterOrderItems(newOrder.Items)

	orderID, err := dbHelpers.InsertIntoOrders(newOrder)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}
	go FindAndPing(newOrder.Mode, newOrder.AddressID, newOrder.UserID, orderID)

	utils.RespondJSON(w, 200, struct {
		OrderID int `json:"order_id"`
	}{orderID})
}

//OrderStatus Get /api/user/order/status/{id}
func OrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}

	status, err := dbHelpers.SelectOrderStatus(orderID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, status)

}

//OrderInfo Get /api/user/order/{id}
func OrderInfo(w http.ResponseWriter, r *http.Request) {
	userID := middlewares.UserContext(r).ID
	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}

	orderDetails, err := dbHelpers.SelectOrder(orderID, userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	orderDetails.Address, err = dbHelpers.SelectAddressWithID(userID, orderDetails.AddressID, false)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, orderDetails)
}

//AllPastOrder Get /api/user/order
func AllPastOrder(w http.ResponseWriter, r *http.Request) {
	userID := middlewares.UserContext(r).ID
	offset, limit, err := utils.GetOffsetLimit(r)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid value for offset or limit")
		return
	}
	allOrder, err := dbHelpers.SelectAllPastOrders(userID, offset, limit)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, allOrder)
}

//CancelOrder Delete /api/user/order/{id}
func CancelOrder(w http.ResponseWriter, r *http.Request) {
	userID := middlewares.UserContext(r).ID
	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}
	isProcessing, err := dbHelpers.SelectOrderStatus(orderID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}

	status := models.Cancelled
	err = dbHelpers.UpdateOrders(orderID, nil, nil, nil, nil, &status, &userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}

	if isProcessing.Status != models.Processing {
		flagCount, status, err := dbHelpers.GetFlagCountAndLastOrderStatus(userID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
			return
		}
		if status == models.Cancelled {
			err := dbHelpers.SetFlag(flagCount+1, userID)
			if err != nil {
				utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
				return
			}
		}
	}

	go func() {
		order, err := dbHelpers.SelectOrder(orderID, userID)
		if err != nil {
			logrus.Error(err)
		}
		address, err := dbHelpers.SelectAddressByOrderId(orderID)
		if err != nil {
			logrus.Error(err)
			return
		}
		staffID := order.StaffID
		if staffID.Valid {
			firebase.OrderStatusUpdateNotification(int64(staffID.Int), orderID, models.Cancelled, address.AddressData)
		}
	}()
	utils.RespondJSON(w, http.StatusOK, models.Response{
		Success: true,
	})

}

// PostScheduledOrder creates new scheduled order(for both cart/delivery mode)
func PostScheduledOrder(w http.ResponseWriter, r *http.Request) {
	userID := middlewares.UserContext(r).ID

	newOrder := models.ScheduledOrder{
		UserID: userID,
	}
	if err := utils.ParseBody(r.Body, &newOrder); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}

	if len(newOrder.Items) == 0 {
		newOrder.Mode = models.CartMode
	} else {
		newOrder.Mode = models.DeliveryMode
	}

	newOrder.Items = utils.FilterOrderItems(newOrder.Items)

	addressLocation, err := dbHelpers.SelectAddressWithID(newOrder.UserID, newOrder.AddressID, false)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid address id")
		return
	}
	smId, err := dbHelpers.StoreManagerNearMe(addressLocation.Lat, addressLocation.Long)
	if err != nil {
		utils.RespondError(w, http.StatusNotAcceptable, err, err.Error(), err.Error())
		return
	}
	newOrder.StoreMangerID = smId

	scheduledOrderID, err := dbHelpers.InsertScheduledOrder(newOrder)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to insert scheduled order")
		return
	}

	scheduledOrder, err := dbHelpers.GetScheduledOrderById(scheduledOrderID, newOrder.UserID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get scheduled order details")
		return
	}

	//todo start go routine here to ping all cart-boys/delivery-boys, according to order mode

	utils.RespondJSON(w, http.StatusCreated, scheduledOrder)
}

// GetAllScheduledOrders returns all scheduled orders for a user
func GetAllScheduledOrders(w http.ResponseWriter, r *http.Request) {
	userID := middlewares.UserContext(r).ID

	scheduledOrder, err := dbHelpers.GetAllScheduledOrders(userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get scheduled order details")
		return
	}

	utils.RespondJSON(w, http.StatusOK, scheduledOrder)
}

// GetScheduledOrder returns details of a particular scheduled order
func GetScheduledOrder(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	userID := userCtx.ID

	orderID, err := utils.StringToInt(chi.URLParam(r, "id"))

	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to convert given orderId to int")
		return
	}

	scheduledOrder, err := dbHelpers.GetScheduledOrderById(orderID, userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get scheduled order details")
		return
	}

	utils.RespondJSON(w, http.StatusOK, scheduledOrder)
}

// ArchiveScheduledOrder archives a given scheduled order
func ArchiveScheduledOrder(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	userID := userCtx.ID

	orderID, err := utils.StringToInt(chi.URLParam(r, "id"))

	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to convert given orderId to int")
		return
	}

	err = dbHelpers.ArchiveScheduledOrder(orderID, userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to archive given scheduled order")
		return
	}

	utils.RespondJSON(w,200,models.Response{
		Success: true,
	})
}

//ConfirmOrderDelivery update order with staff-rating
func ConfirmOrderDelivery(w http.ResponseWriter, r *http.Request) {
	userID := middlewares.UserContext(r).ID
	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}
	rating := models.ConfirmOrder{}
	err = utils.ParseBody(r.Body, &rating)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse request body")
		return
	}
	staffRating := rating.StaffRating
	err = dbHelpers.UpdateOrders(orderID, nil, nil, nil, &staffRating, nil, &userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w,200,models.Response{
		Success: true,
	})
}

//OrderInfoForStaff Returns an object of order detail for given order id
func OrderInfoForStaff(w http.ResponseWriter, r *http.Request) {
	staffID := middlewares.UserContext(r).ID
	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}

	orderDetails, err := dbHelpers.GetOrderByID(orderID, staffID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	userInfo, err := dbHelpers.GetUserById(orderDetails.UserID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get user info")
		return
	}
	orderDetails.UserImage = userInfo.ProfileImageLink
	utils.RespondJSON(w, 200, orderDetails)
}

func GetActiveOrdersForUser(w http.ResponseWriter, r *http.Request) {
	userID := middlewares.UserContext(r).ID
	AllOrder, err := dbHelpers.SelectActiveOrders(userID, 0, 1)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, AllOrder)
}

func DisputeOrder(w http.ResponseWriter, r *http.Request) {
	userID := middlewares.UserContext(r).ID
	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}

	err = dbHelpers.InsertDisputedOrder(orderID, userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}

	utils.RespondJSON(w,200,models.Response{
		Success: true,
	})

}

func GetScheduledOrders(w http.ResponseWriter, r *http.Request) {
	smId := middlewares.UserContext(r).ID
	orders, err := dbHelpers.GetAllScheduledOrdersForSm(smId)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, orders)
}

func GetOngoingOrder(w http.ResponseWriter, r *http.Request) {
	smId := middlewares.UserContext(r).ID
	orders, err := dbHelpers.GetOnGoingOrder(smId)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, orders)

}
