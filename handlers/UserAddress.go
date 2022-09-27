package handlers

import (
	"github.com/RemoteState/yourdaily-server/dbHelpers"
	"github.com/RemoteState/yourdaily-server/middlewares"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"github.com/go-chi/chi"
	"net/http"
	"strconv"
)

//PostAddress POST::/api/user/address
func PostAddress(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	userID := userCtx.ID

	addressData := models.Address{
		UserID:     userID,
		IsDefault:  false,
		AddressTag: models.DefaultAddressTag,
	}

	if err := utils.ParseBody(r.Body, &addressData); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse req body")
		return
	}

	if allAddress, err := dbHelpers.SelectAllAddress(userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to insert data into database")
		return
	} else if len(allAddress) == 0 {
		addressData.IsDefault = true
	}

	if err := dbHelpers.InsertAddress(addressData); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to insert data into database")
		return
	}
	addresses, err := dbHelpers.SelectAllAddress(userID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, addresses)
}

//GetAllAddress GET::/api/user/address
func GetAllAddress(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)

	userID := userCtx.ID

	addresses, err := dbHelpers.SelectAllAddress(userID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, addresses)
}

//PutAddress PUT::/api/user/address/{id}
func PutAddress(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	userID := userCtx.ID

	addID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid address ID")
		return
	}
	addressData := models.Address{
		ID:         addID,
		UserID:     userID,
		AddressTag: models.DefaultAddressTag,
	}
	if err := utils.ParseBody(r.Body, &addressData); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse req body")
		return
	}

	if err = dbHelpers.UpdateAddress(addressData); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "unable to update address in database")
		return
	}
	addresses, err := dbHelpers.SelectAllAddress(userID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, addresses)
}

//ArchiveAddress DELETE::/api/user/address/{id}
func ArchiveAddress(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	userID := userCtx.ID

	addID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid address ID")
		return
	}

	if err := dbHelpers.ArchiveAddress(addID, userID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	addresses, err := dbHelpers.SelectAllAddress(userID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, addresses)
}

func GetAddressByID(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	userID := userCtx.ID

	addID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "invalid address ID")
		return
	}

	address, err := dbHelpers.SelectAddressWithID(userID, addID, false)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), err.Error())
		return
	}

	utils.RespondJSON(w, 200, address)
}
