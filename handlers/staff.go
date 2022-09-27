package handlers

import (
	"errors"
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

func AllActiveOrders(w http.ResponseWriter, r *http.Request) {
	ctx := middlewares.UserContext(r)
	staffID := ctx.ID
	mode := ctx.AllowedMode
	if mode == "" {
		utils.RespondError(w, http.StatusUnauthorized, errors.New("empty mode string"), "invalid user permission")
		return
	}

	AllOrder, err := dbHelpers.SelectAllActiveOrdersForStaff(staffID, mode)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, AllOrder)

}

//PutLocation updates the location of
func PutLocation(w http.ResponseWriter, r *http.Request) {
	ctx := middlewares.UserContext(r)
	staffID := ctx.ID
	loc := models.GeoLocation{}
	err := utils.ParseBody(r.Body, &loc)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse request body")
		return
	}

	err = dbHelpers.UpdateLocation(staffID, loc)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to update location")
		return
	}

	utils.RespondJSON(w,200,models.Response{
		Success: true,
	})

}

//PutAcceptOrder handle put req to accept order from staff
func PutAcceptOrder(w http.ResponseWriter, r *http.Request) {
	ctx := middlewares.UserContext(r)
	staffID := ctx.ID
	mode := ctx.AllowedMode
	if mode == "" {
		utils.RespondError(w, http.StatusUnauthorized, errors.New("empty mode string"), "invalid user permission")
		return
	}
	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid order id")
		return
	}
	activeOrder, err := dbHelpers.SelectAllActiveOrdersForStaff(staffID, mode)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "something went wrong")
		return
	}
	if len(activeOrder) > models.OrderAcceptanceLimit {
		utils.RespondJSON(w, http.StatusOK, models.OrderAccept{
			OrderID:  orderID,
			Accepted: false,
			Message:  "order acceptance limit reached",
		})
		return
	}
	orderStatus, err := dbHelpers.SelectOrderStatus(orderID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "something went wrong")
		return
	}
	if orderStatus.Status != models.Processing {
		utils.RespondJSON(w, http.StatusOK, models.OrderAccept{
			OrderID:  orderID,
			Accepted: false,
			Message:  "Order Already accepted",
		})
		return
	}
	status := models.Accepted
	err = dbHelpers.UpdateOrders(orderID, &staffID, nil, nil, nil, &status, nil)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "something went wrong")
		return
	}

	go func() {
		order, err := dbHelpers.GetOrderByID(orderID, staffID)
		if err != nil {
			logrus.Error(err)
			return
		}
		userID := order.UserID
		firebase.OrderStatusUpdateNotification(int64(userID), orderID, models.Accepted, "")
	}()

	//staffOrder, err := dbHelpers.SelectAllActiveOrdersForStaff(staffID, mode)
	//if err != nil {
	//	utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "something went wrong")
	//	return
	//}
	utils.RespondJSON(w, http.StatusOK, models.OrderAccept{
		OrderID:  orderID,
		Accepted: true,
		Message:  "Order Accepted successfully",
	})
}

//PutOutForDelivery Updates the order status to out for delivery
func PutOutForDelivery(w http.ResponseWriter, r *http.Request) {
	staffID := middlewares.UserContext(r).ID

	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid order id")
		return
	}

	check, err := dbHelpers.CheckOrderStatus(staffID, orderID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to fetch order status")
		return
	}

	if check > 0 {
		err := fmt.Errorf("complete previous order")
		utils.RespondError(w, http.StatusNotAcceptable, err, err.Error(), err.Error())
		return
	}
	status := models.OutForDelivery
	err = dbHelpers.UpdateOrders(orderID, nil, nil, nil, nil, &status, nil)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "something went wrong")
		return
	}
	go func() {
		order, err := dbHelpers.GetOrderByID(orderID, staffID)
		if err != nil {
			logrus.Error(err)
		}
		userID := order.UserID
		firebase.OrderStatusUpdateNotification(int64(userID), orderID, models.OutForDelivery, "")
	}()
	utils.RespondJSON(w,200,models.Response{
		Success: true,
	})
}

//PostVerifyOTP Post req to very verify otp
func PostVerifyOTP(w http.ResponseWriter, r *http.Request) {
	staffID := middlewares.UserContext(r).ID
	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid order id")
		return
	}

	verifyOrder := models.VerifyOrder{}
	err = utils.ParseBody(r.Body, &verifyOrder)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse json body")
		return
	}

	storedOTP, err := dbHelpers.GetOTP(orderID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "something went wrong")
		return
	}
	status := models.Delivered
	if verifyOrder.OTP == storedOTP {
		err := dbHelpers.UpdateOrders(orderID, nil, &verifyOrder.Amount, &verifyOrder.UserRating, nil, &status, nil)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to update order")
			return
		}
		err = dbHelpers.DeleteOTP(orderID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "Unable to delete otp")
			return
		}

		go func() {
			order, err := dbHelpers.GetOrderByID(orderID, staffID)
			if err != nil {
				logrus.Error(err)
			}
			userID := order.UserID
			firebase.OrderStatusUpdateNotification(int64(userID), orderID, models.Delivered, "")
		}()

		utils.RespondJSON(w,200,models.Response{
			Success: true,
		})
		return
	} else {
		utils.RespondError(w, http.StatusUnauthorized, fmt.Errorf("invalid otp"), "invalid  otp")
		return
	}

}

//GetOrderHistory returns order history of all delivered orders for given staff
func GetOrderHistory(w http.ResponseWriter, r *http.Request) {
	staffID := middlewares.UserContext(r).ID
	offset, limit, err := utils.GetOffsetLimit(r)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid value for offset or limit")
		return
	}

	history, err := dbHelpers.AllCompletedOrders(staffID, offset, limit)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "Unable to fetch data")
		return
	}
	utils.RespondJSON(w, 200, history)
}

// RegisterStaff creates new guest user[for staff members- cart/delivery boy]
// once approved by store-manager, these guest will get (user, cartboy) OR (user, deliv-boy) permissions
func RegisterStaff(w http.ResponseWriter, r *http.Request) {

	fireBaseToken := r.Header.Get("Authorization")

	jwt, err := firebase.FireAuthInstance.VerifyToken(fireBaseToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	authId, err := firebase.GetAuthId(jwt)
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	reqBody := struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}

	// check if phone exists in db, if yes, receives userID from db else 0
	exist, err := dbHelpers.IsUserExist(authId)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to find user")
	}

	userID := exist

	// user does not exist -> create new
	if exist == 0 {
		userID, err = dbHelpers.InsertUser(reqBody.Name, reqBody.Phone, authId, models.Guest)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "failed to create user")
			return
		}
	}

	user, err := dbHelpers.GetUserById(userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to get user details")
		return
	}

	utils.RespondJSON(w, http.StatusOK, user)
}

func GetNewOrders(w http.ResponseWriter, r *http.Request) {
	staffID := middlewares.UserContext(r).ID
	mode := middlewares.UserContext(r).AllowedMode

	newOrders, err := dbHelpers.GetNewOrders(staffID, mode)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch new orders")
		return
	}
	locationOfStaff, err := dbHelpers.GetStaffLocationByID(staffID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch staff location")
		return
	}

	newOrderForStaff := make([]models.OrderNotification, 0)
	for _, v := range newOrders {
		dist := utils.GeoDistance(v.UserLong.Float64, v.UserLat.Float64, locationOfStaff.Long.Float64, locationOfStaff.Lat.Float64)
		if dist < float64(models.RadiusForSearch) {
			order := models.OrderNotification{
				OrderID:     v.OrderId,
				AddressData: v.UserAddressData,
				Lat:         v.UserLat.Float64,
				Long:        v.UserLong.Float64,
				ExpireTime:  (v.DeliveryTime.Time.Add(30 * time.Second)).Unix(),
			}
			newOrderForStaff = append(newOrderForStaff, order)
		}
	}

	utils.RespondJSON(w, http.StatusOK, newOrderForStaff)
}

func RejectOrderForStaff(w http.ResponseWriter, r *http.Request) {
	staffID := middlewares.UserContext(r).ID
	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		err := fmt.Errorf("invalid order id")
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
		return
	}
	err = dbHelpers.RejectOrder(staffID, orderID)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w,200,models.Response{
		Success: true,
	})
}
