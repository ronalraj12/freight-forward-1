package handlers

import (
	"database/sql"
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
)

// Register creates new user if does not exists and return auth token
func Register(w http.ResponseWriter, r *http.Request) {

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

	//TODO if user exits for the above authId in our DB then ok
	// if user does not exits in our DB then insert name, phone and authId

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
		userID, err = dbHelpers.InsertUser(reqBody.Name, reqBody.Phone, authId, models.DefaultUser)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to create user")
			return
		}
	}

	user, err := dbHelpers.GetUserById(userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get user details")
		return
	}

	utils.RespondJSON(w, http.StatusOK, user)
}

// GetUserInfo returns user info
func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	utils.RespondJSON(w, http.StatusOK, userCtx)
}

// UpdateUserInfo updates & returns updated user info
func UpdateUserInfo(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	userID := userCtx.ID

	reqBody := struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}
	exist, err := dbHelpers.CheckIfEmailIsRegisteredToSomeoneElse(reqBody.Email, userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	if exist != 0 {
		err := fmt.Errorf("email already registered %s ", reqBody.Email)
		utils.RespondError(w, http.StatusConflict, err, err.Error(), err.Error())
		return
	}
	if err := dbHelpers.ModifyUser(reqBody.Name, reqBody.Email, userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update user details")
		return
	}

	user, err := dbHelpers.GetUserById(userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get user details")
		return
	}
	utils.RespondJSON(w, http.StatusCreated, user)
}

func IsPhoneExisting(w http.ResponseWriter, r *http.Request) {
	reqBody := struct {
		Phone string `json:"phone"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}

	userExists := true
	if _, err := dbHelpers.IsPhoneExist(reqBody.Phone); err != nil {
		if err == sql.ErrNoRows {
			userExists = false
			//fmt.Println("updated", userExists)
		} else {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to check if phone exists")
			return
		}
	}
	//fmt.Println(userExists)

	response := struct {
		Exists bool `json:"userExists"`
	}{
		Exists: userExists,
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

func UpdateFcmToken(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	userID := userCtx.ID

	reqBody := struct {
		FCMToken string `json:"fcmToken"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}

	if reqBody.FCMToken == "" {
		utils.RespondError(w, http.StatusBadRequest, errors.New("fcm token is empty"), "Invalid fcm token")
		return
	}

	if err := dbHelpers.ModifyFcmToken(reqBody.FCMToken, userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update fcm token")
		return
	}

	utils.RespondJSON(w, 200, models.Response{Success: true})
}

func UploadImage(w http.ResponseWriter, r *http.Request) {
	file, fileBytes, downloadedFileName, err := utils.ReadFromFile(r, string(models.ProfileImage))
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

	userCtx := middlewares.UserContext(r)
	userID := userCtx.ID

	if err := dbHelpers.StoreProfileImage(models.BucketLink, uploadedFileName, models.ProfileImage, userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in storing uploaded file info")
		return
	}

	imageInfo, err := dbHelpers.GetImageInfoByUserID(userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in generating image link")
		return
	}

	url, err := firebase.GetURL(imageInfo)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in getting image URL")
		return
	}

	response := struct {
		ImageURL string `json:"imageURL"`
	}{
		ImageURL: url,
	}
	utils.RespondJSON(w, http.StatusOK, response)
}

func GetStaffByID(w http.ResponseWriter, r *http.Request) {
	staffID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}

	user, err := dbHelpers.GetUserById(staffID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get user details")
		return
	}
	user.Rating, err = dbHelpers.GetStaffRating(staffID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get user rating")
		return
	}
	utils.RespondJSON(w, 200, user)
}

func GetActiveDiscount(w http.ResponseWriter, r *http.Request) {
	activeOffer, err := dbHelpers.GetActiveOffer()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get active offer info")
		return
	}

	utils.RespondJSON(w, http.StatusOK, activeOffer)
}

func GetAdminContactInfo(w http.ResponseWriter, r *http.Request) {
	//contact, err := dbHelpers.AdminContactInfo()
	//if err != nil {
	//	utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
	//	return
	//}
	utils.RespondJSON(w, 200, struct {
		Phone string `json:"phone"`
		Email string `json:"email"`
	}{
		Phone: "+918750633567",
		Email: "customercare@yourdaily.co.in",
	})

}
