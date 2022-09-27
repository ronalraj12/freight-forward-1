package handlers

import (
	"database/sql"
	"fmt"
	"github.com/RemoteState/yourdaily-server/dbHelpers"
	"github.com/RemoteState/yourdaily-server/firebase"
	"github.com/RemoteState/yourdaily-server/middlewares"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	user        = "user"
	cartBoy     = "cart-boy"
	DeliveryBoy = "delivery-boy"
)

//LoginStoreManager login for  store manager
func LoginStoreManager(w http.ResponseWriter, r *http.Request) {
	var smCred models.StoreMangerCred
	if err := utils.ParseBody(r.Body, &smCred); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse req body")
		return
	}
	smCred.Password = utils.HashString(smCred.Password)
	logrus.Info(smCred.Password)

	user, err := dbHelpers.GetStoreManagerByEmail(smCred.Email)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}

	if user.Permission != models.StoreManager {
		utils.RespondError(w, http.StatusUnauthorized, errors.New("given email does not belong to a store-manager"), "Authentication failed!")
		return
	}

	user.Password = utils.HashString(user.Password) // todo need to remove this, save pwd as hash using super-admin routes in future
	if user.Password != smCred.Password {
		err := fmt.Errorf("invaid credentials")
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
		return
	}
	token, err := utils.GenerateJWT(user.ID, user.Email.String)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, struct {
		Token string `json:"Authorization"`
	}{
		Token: token,
	})
	logrus.Print(smCred.Password)
}

//DashBoardStats return an object filled with all the sm dashboard screen
func DashBoardStats(w http.ResponseWriter, r *http.Request) {

	smID := middlewares.UserContext(r).ID
	var (
		stats models.DashBoardStats
		egp   = new(errgroup.Group)
	)

	egp.Go(func() error {
		var err error
		guest, err := dbHelpers.GetAllGuest()
		if err != nil {
			return err
		}
		stats.UnapprovedStaff = len(guest)
		return nil

	})

	egp.Go(func() error {
		var err error
		userIds, err := dbHelpers.GerUserCount()
		stats.UserCount = len(userIds)
		return err
	})

	egp.Go(func() error {
		var err error
		cartBoyIds, err := dbHelpers.GetStaffCount(cartBoy)
		stats.CartBoyCount = len(cartBoyIds)
		return err
	})
	egp.Go(func() error {
		var err error
		deliveryBoydIds, err := dbHelpers.GetStaffCount(DeliveryBoy)
		stats.DeliveryBoyCount = len(deliveryBoydIds)
		return err
	})

	egp.Go(func() error {
		var err error
		stats.TotalItems, err = dbHelpers.GetItemCount()
		return err
	})
	egp.Go(func() error {
		var err error
		stats.UnassignedOrders, err = dbHelpers.GetOrderCountForStatus(models.Processing)
		return err
	})
	egp.Go(func() error {
		var err error
		stats.DeniedOrder, err = dbHelpers.GetOrderCountForStatus(models.Declined)
		return err
	})

	egp.Go(func() error {
		var err error
		stats.UnassignedOrders, err = dbHelpers.GetUnassignedOrderCount()
		return err
	})

	egp.Go(func() error {
		var err error
		onGoingOrderID, err := dbHelpers.GetOnGoingOrder(smID)
		stats.OnGoingOrder = len(onGoingOrderID)
		return err
	})

	egp.Go(func() error {
		var err error
		stats.ActiveUsers, err = dbHelpers.GetActiveUsers()
		return err
	})

	egp.Go(func() error {
		var err error
		stats.BookingForLastWeek, err = dbHelpers.GetLastWeekBookingCount()
		return err
	})
	egp.Go(func() error {
		var err error
		DisputedOrder, err := dbHelpers.GetAllDisputedOrders()
		stats.DisputedOrder = len(DisputedOrder)
		return err
	})

	egp.Go(func() error {
		var err error
		scheduledOrder, err := dbHelpers.GetAllScheduledOrdersForSm(smID)
		stats.ScheduledOrder = len(scheduledOrder)
		return err

	})

	err := egp.Wait()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "something went wrong")
		return
	}

	utils.RespondJSON(w, 200, stats)

}

//GetOrderTypeGraphData return a count of order based on order types
func GetOrderTypeGraphData(w http.ResponseWriter, r *http.Request) {

	days, err := strconv.Atoi(chi.URLParam(r, "days"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid value for days")
		return
	}
	orderStats, err := dbHelpers.GetNSGStats(days)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, orderStats)
}

//GetOrderAcceptGraphData return the graph data for accepted declined orders
func GetOrderAcceptGraphData(w http.ResponseWriter, r *http.Request) {
	days, err := strconv.Atoi(chi.URLParam(r, "days"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid value for days")
		return
	}
	orderStats, err := dbHelpers.GetOrderAcceptedStats(days)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, orderStats)
}

// ApproveStaff adds cart-boy/delivery-boy permission to a given guest user
func ApproveStaff(w http.ResponseWriter, r *http.Request) {
	reqBody := struct {
		UserID     int                   `json:"userID"`
		Permission models.UserPermission `json:"permission"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}

	err := dbHelpers.UpdateStaffPermission(reqBody.UserID, reqBody.Permission)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to assign permission to given user")
		return
	}

	user, err := dbHelpers.GetUserById(reqBody.UserID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get staff details")
		return
	}

	user.Permissions, err = dbHelpers.UserPermissionById(reqBody.UserID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get staff's updated permissions")
		return
	}

	utils.RespondJSON(w, http.StatusOK, user)
}

// ArchiveStaff archives a given staff(cart/delivery boy)
func ArchiveStaff(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid user ID")
		return
	}

	if err := dbHelpers.ArchiveStaffByUserID(userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "failed to archive staff member")
		return
	}
	utils.RespondJSON(w,200,models.Response{
		Success: true,
	})
}

func GetAllStaffStats(w http.ResponseWriter, r *http.Request) {

	staffType := chi.URLParam(r, "staffType")
	if !(staffType == cartBoy || staffType == DeliveryBoy) {
		err := fmt.Errorf("invalide staffType recived as url param")
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
		return
	}

	staffStats, err := dbHelpers.GetStaffStats(staffType, nil)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}

	for i := range staffStats {
		imageInfo, err := dbHelpers.GetImageInfoByUserID(staffStats[i].ID)
		if err != nil {
			logrus.Errorf("Failed to get staff profile image info with error: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if imageInfo != nil {
			imageURL, err := firebase.GetURL(imageInfo)
			if err != nil {
				logrus.Errorf("Failed to get image url with error: %+v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			staffStats[i].ProfileImageLink = imageURL
		}
	}
	utils.RespondJSON(w, 200, staffStats)
}

func GetAllUserStats(w http.ResponseWriter, r *http.Request) {

	staffStats, err := dbHelpers.GetAllUserStats(nil)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	for i := range staffStats {
		imageInfo, err := dbHelpers.GetImageInfoByUserID(staffStats[i].ID)
		if err != nil {
			logrus.Errorf("Failed to get staff profile image info with error: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if imageInfo != nil {
			imageURL, err := firebase.GetURL(imageInfo)
			if err != nil {
				logrus.Errorf("Failed to get image url with error: %+v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			staffStats[i].ProfileImageLink = imageURL
		}
	}
	utils.RespondJSON(w, 200, staffStats)
}

func GetAllOrdersWithStatus(w http.ResponseWriter, r *http.Request) {
	orderType := chi.URLParam(r, "orderType")
	var status models.OrderStatus
	if orderType == "unassigned" {
		status = models.Processing
	} else if orderType == "denied" {
		status = models.Declined
	} else {
		err := fmt.Errorf("invalide OrderType recived as url param")
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
		return
	}

	AllOrders, err := dbHelpers.GetOrdersForStatus(status)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, AllOrders)
}

func GetUnapprovedStaff(w http.ResponseWriter, r *http.Request) {
	guests, err := dbHelpers.GetAllGuest()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get unapproved staff list")
		return
	}
	utils.RespondJSON(w, http.StatusOK, guests)
}

func GetAllDisputedOrders(w http.ResponseWriter, r *http.Request) {
	disOrder, err := dbHelpers.GetAllDisputedOrders()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, disOrder)
}
func GetAllDisputedOrderInfo(w http.ResponseWriter, r *http.Request) {
	orderId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, 500, err, err.Error(), "invalid order id")
		return
	}
	disOrder, err := dbHelpers.GetDisputedOrderInfo(orderId)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, disOrder)
}
func GetStaffStatsByID(w http.ResponseWriter, r *http.Request) {
	staffID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "Invalid user ID")
		return
	}
	staffType := chi.URLParam(r, "staffType")
	if !(staffType == cartBoy || staffType == DeliveryBoy) {
		err := fmt.Errorf("invalide staffType recived as url param")
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
		return
	}

	staffStats, err := dbHelpers.GetStaffStats(staffType, &staffID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}

	for i := range staffStats {
		imageInfo, err := dbHelpers.GetImageInfoByUserID(staffStats[i].ID)
		if err != nil {
			logrus.Errorf("Failed to get staff profile image info with error: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if imageInfo != nil {
			imageURL, err := firebase.GetURL(imageInfo)
			if err != nil {
				logrus.Errorf("Failed to get image url with error: %+v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			staffStats[i].ProfileImageLink = imageURL
		}
	}
	if len(staffStats) == 0 {
		err := fmt.Errorf("invalid staff id recived ,unauthorized access")
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, staffStats)
}
func GetUserStatsByID(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "Invalid user ID")
		return
	}
	userStats, err := dbHelpers.GetAllUserStats(&userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	for i := range userStats {
		imageInfo, err := dbHelpers.GetImageInfoByUserID(userStats[i].ID)
		if err != nil {
			logrus.Errorf("Failed to get staff profile image info with error: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if imageInfo != nil {
			imageURL, err := firebase.GetURL(imageInfo)
			if err != nil {
				logrus.Errorf("Failed to get image url with error: %+v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			userStats[i].ProfileImageLink = imageURL
		}
	}
	if len(userStats) == 0 {
		err := fmt.Errorf("invalid user id recived ,unauthorized access")
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, userStats)
}

func GetNearbyStaffList(w http.ResponseWriter, r *http.Request) {
	orderID, err := utils.StringToInt(chi.URLParam(r, "orderId"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "Invalid order ID")
		return
	}

	// get order details using order id
	orderInfo, err := dbHelpers.GetOrderInfo(orderID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get order info for given id")
		return
	}

	// get user location details
	userLocation, err := dbHelpers.SelectAddressWithID(orderInfo.UserID, orderInfo.AddressID, false)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get user location")
		return
	}

	// all staff location details
	allStaffLocations, err := dbHelpers.SelectLocationOfStaff(orderInfo.Mode)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get staff locations")
		return
	}

	// get nearby staff details
	nearbyStaff := make([]models.User, 0)
	for _, v := range allStaffLocations {
		dist := utils.GeoDistance(v.Long.Float64, v.Lat.Float64, userLocation.Long, userLocation.Lat)
		if dist < float64(models.RadiusForSearch) {
			staff, err := dbHelpers.GetUserById(v.StaffID.Int)
			if err != nil {
				utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get staff details")
				return
			}

			permissions, err := dbHelpers.UserPermissionById(staff.ID)
			if err != nil {
				utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get staff permissions")
				return
			}
			staff.Permissions = permissions

			imageInfo, err := dbHelpers.GetImageInfoByUserID(staff.ID)
			if err != nil {
				utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get staff image info")
				return
			}

			if imageInfo != nil {
				imageURL, err := firebase.GetURL(imageInfo)
				if err != nil {
					utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get staff image URL")
					return
				}
				staff.ProfileImageLink = imageURL
			}
			nearbyStaff = append(nearbyStaff, *staff)
		}
	}
	utils.RespondJSON(w, http.StatusOK, nearbyStaff)
}

func AssignOrderToStaff(w http.ResponseWriter, r *http.Request) {
	reqBody := struct {
		OrderID int `json:"orderId"`
		StaffID int `json:"staffId"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}

	status, err := dbHelpers.GetCurrentOrderStatus(reqBody.OrderID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get current order status")
		return
	}

	if status != models.Processing {
		invalidStatus := errors.Errorf("order status found '%v', but should be 'processing'", status)
		utils.RespondError(w, http.StatusBadRequest, invalidStatus, "Failed to assign given order")
		return
	} else {
		status = models.Accepted
	}

	// update order
	err = dbHelpers.UpdateOrders(reqBody.OrderID, &reqBody.StaffID, nil, nil, nil, &status, nil)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to assign given order to given staff")
		return
	}

	// todo notify staff

	response := struct {
		Assigned bool `json:"assigned"`
	}{
		Assigned: true,
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

func MarkAsResolved(w http.ResponseWriter, r *http.Request) {
	smId := middlewares.UserContext(r).ID
	orderId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, 500, err, err.Error(), "invalid order id")
		return
	}
	err = dbHelpers.MarkAsResolved(orderId, smId)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to mark order as resolved,try again after some time")
	}
	w.WriteHeader(200)
}

func RejectStaff(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid guest user ID")
		return
	}

	if err := dbHelpers.RemoveGuest(userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "failed to remove guest permission")
		return
	}

	// todo we will now archive user (now/after a fixed period/after a number of requests)
	// also need to logout this guest user if not archived

	response := struct {
		Rejected bool `json:"rejected"`
	}{
		Rejected: true,
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// AddImageOfGivenType adds new image of type 'item' or 'offer'
func AddImageOfGivenType(w http.ResponseWriter, r *http.Request) {
	imageType := chi.URLParam(r, "imageType")
	if imageType == "" {
		utils.RespondError(w, http.StatusBadRequest, nil, "Image type can't be empty!")
		return
	}

	if !models.IsValidImageType(imageType) {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid image type")
	}

	file, fileBytes, downloadedFileName, err := utils.ReadFromFile(r, imageType)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in reading image file")
		return
	}

	defer func() {
		err = file.Close()
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed in closing file")
			return
		}
	}()

	uploadedFileName, err := firebase.UploadToFirebase(fileBytes, downloadedFileName)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in uploading to firebase")
		return
	}

	imageID, err := dbHelpers.StoreImageInfo(models.BucketLink, uploadedFileName, imageType)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in storing uploaded file info")
		return
	}

	response := struct {
		ImageID int `json:"imageId"`
	}{
		ImageID: imageID,
	}
	utils.RespondJSON(w, http.StatusOK, response)
}

func CreateNewOffer(w http.ResponseWriter, r *http.Request) {
	reqBody := struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Discount    int      `json:"discount"`
		ImageID     null.Int `json:"imageId"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}

	if err := dbHelpers.CreateNewOffer(reqBody.Title, reqBody.Description, reqBody.Discount, reqBody.ImageID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to create new offer entry")
		return
	}

	activeOffer, err := dbHelpers.GetActiveOffer()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get active offer info")
		return
	}

	imageInfo, err := dbHelpers.GetImageInfoByOfferID(activeOffer.ID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get image info")
		return
	}
	var imageURL string
	if imageInfo != nil {
		imageURL, err = firebase.GetURL(imageInfo)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get image URL")
			return
		}
	}
	activeOffer.ImageURL = imageURL
	utils.RespondJSON(w, http.StatusCreated, activeOffer)
}

func ArchiveActiveOffer(w http.ResponseWriter, r *http.Request) {
	if err := dbHelpers.ArchiveActiveOffer(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to archive offer")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func GetActiveOffer(w http.ResponseWriter, r *http.Request) {
	activeOffer, err := dbHelpers.GetActiveOffer()
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusOK)
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get active offer info")
		return
	}
	imageInfo, err := dbHelpers.GetImageInfoByOfferID(activeOffer.ID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get image info")
		return
	}
	var imageURL string
	if imageInfo != nil {
		imageURL, err = firebase.GetURL(imageInfo)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get image URL")
			return
		}
	}
	activeOffer.ImageURL = imageURL
	utils.RespondJSON(w, http.StatusOK, activeOffer)
}

func GetNewOrderForStoreManger(w http.ResponseWriter, r *http.Request) {
	newOrder, err := dbHelpers.GetNewOrdersForSm()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, newOrder)
}

func DownloadScheduledOrders(w http.ResponseWriter, r *http.Request) {
	reqBody := struct {
		StartDate time.Time `json:"startDate"`
		EndDate   time.Time `json:"endDate"`
	}{}

	err := utils.ParseBody(r.Body, &reqBody)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse req body")
		return
	}
	reqBody.StartDate = reqBody.StartDate.Add(24 * time.Hour)
	reqBody.EndDate = reqBody.EndDate.Add(24 * time.Hour)
	orders, err := dbHelpers.GetScheduledOrderInRange(reqBody.StartDate, reqBody.EndDate)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to fetch scheduled orders")
		return
	}

	fileName, err := utils.CreateCsvOfOrders(orders)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to create csv file for orders")
		return
	}
	file, err := os.Open(fileName)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to download csv file for orders")
		return
	}
	defer file.Close()
	w.Header().Set("Content-Disposition", "attachment; filename=scheduledOrdersList.csv")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	_, err = io.Copy(w, file)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to download csv file for orders")
		return
	}

	if err = os.Remove(fileName); err != nil {
		logrus.Errorf("unable to delete file %v", err)
		return
	}
}

func GetOrders(w http.ResponseWriter, r *http.Request) {
	smId := middlewares.UserContext(r).ID

	reqBody := struct {
		StartDate time.Time `json:"startDate"`
		EndDate   time.Time `json:"endDate"`
	}{}

	err := utils.ParseBody(r.Body, &reqBody)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse req body")
		return
	}
	reqBody.StartDate = reqBody.StartDate.Add(24 * time.Hour)
	reqBody.EndDate = reqBody.EndDate.Add(24 * time.Hour)

	orders, err := dbHelpers.GetOrderInDateRange(reqBody.StartDate, reqBody.EndDate, smId)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, orders)
}

func DownloadUserStats(w http.ResponseWriter, r *http.Request) {

	userStats, err := dbHelpers.GetAllUserStats(nil)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to fetch user details")
		return
	}

	fileName, err := utils.CreateCsvOfUserStats(userStats)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to create csv file for orders")
		return
	}
	file, err := os.Open(fileName)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to download csv file for orders")
		return
	}
	defer file.Close()
	w.Header().Set("Content-Disposition", "attachment; filename=UserStats.csv")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	_, err = io.Copy(w, file)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to download csv file for orders")
		return
	}

	if err = os.Remove(fileName); err != nil {
		logrus.Errorf("unable to delete file %v", err)
		return
	}

}

func DownloadOrderHistory(w http.ResponseWriter, r *http.Request) {
	smId := middlewares.UserContext(r).ID
	reqBody := struct {
		StartDate time.Time `json:"startDate"`
		EndDate   time.Time `json:"endDate"`
	}{}

	err := utils.ParseBody(r.Body, &reqBody)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse req body")
		return
	}
	reqBody.StartDate = reqBody.StartDate.Add(24 * time.Hour)
	reqBody.EndDate = reqBody.EndDate.Add(24 * time.Hour)
	orders, err := dbHelpers.GetOrderInDateRange(reqBody.StartDate, reqBody.EndDate, smId)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to fetch scheduled orders")
		return
	}

	fileName, err := utils.CreateCsvOfOrderHistory(orders)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to create csv file for orders")
		return
	}
	file, err := os.Open(fileName)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to download csv file for orders")
		return
	}
	defer file.Close()
	w.Header().Set("Content-Disposition", "attachment; filename=orderHistory.csv")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	_, err = io.Copy(w, file)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to download csv file for orders")
		return
	}

	if err = os.Remove(fileName); err != nil {
		logrus.Errorf("unable to delete file %v", err)
		return
	}
}

func GetItemsForStoreManager(w http.ResponseWriter, r *http.Request) {
	items, err := dbHelpers.GetItems(true)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get item entries")
		return
	}
	for i := range items {
		imagesInfo, err := dbHelpers.GetImageInfoByItemID(items[i].ID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed in generating image link")
			return
		}

		imageUrls := make([]string, 0)
		for j := range imagesInfo {
			url, err := firebase.GetURL(&imagesInfo[j])
			if err != nil {
				utils.RespondError(w, http.StatusInternalServerError, err, "Failed in getting image URL")
				return
			}
			imageUrls = append(imageUrls, url)
		}
		items[i].ItemImageLinks = imageUrls
	}
	utils.RespondJSON(w, http.StatusOK, items)
}

func UnFlagUser(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid user id")
		return
	}

	err = dbHelpers.SetUnFlag(userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	w.WriteHeader(200)
}

func CancelScheduledOrder(w http.ResponseWriter, r *http.Request) {
	smId := middlewares.UserContext(r).ID
	orderID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid order id")
		return
	}
	userID, err := dbHelpers.ArchiveScheduledOrderWithOrderID(orderID, smId)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}

	go firebase.ScheduledOrderCanceledNotification(userID, orderID, "your order has been canceled by store manager!")
	utils.RespondJSON(w, http.StatusOK, models.Response{
		Success: true,
	})
}

func EnableDisableStaff(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid User id")
		return
	}
	status := chi.URLParam(r, "status")
	enabled := true
	if status == "disable" {
		enabled = false
	}
	err = dbHelpers.EnableDisableStaff(enabled, userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "invalid User id")
		return
	}

	utils.RespondJSON(w, http.StatusOK, models.Response{Success: true})
}

func ChangeStaffRole(w http.ResponseWriter, r *http.Request) {
	var reqBody = struct {
		ID      int    `json:"id"`
		NewRole string `json:"newRole"`
	}{}
	err := utils.ParseBody(r.Body, &reqBody)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable tp parse json body")
	}

	err = dbHelpers.ChangeStaffRole(reqBody.ID, reqBody.NewRole)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "invalid User id")
		return
	}
	utils.RespondJSON(w, http.StatusOK, models.Response{
		Success: true,
	})
}
