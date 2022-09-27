package handlers

import (
	"fmt"
	"github.com/RemoteState/yourdaily-server/dbHelpers"
	"github.com/RemoteState/yourdaily-server/middlewares"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"net/http"
	"strconv"
)

func GetAllMessage(w http.ResponseWriter, r *http.Request) {
	var (
		orderID int
		err     error
	)
	if r.URL.Query().Get("orderID") != "" {
		orderID, err = strconv.Atoi(r.URL.Query().Get("orderID"))
		if err != nil {
			utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
			return
		}
	} else {
		err = fmt.Errorf("invalid orderID")
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error(), err.Error())
		return
	}
	chats, err := dbHelpers.GetAllMessage(orderID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, chats)
}

func PostMessage(w http.ResponseWriter, r *http.Request) {
	Sender := middlewares.UserContext(r).ID

	chat := models.Chat{}
	err := utils.ParseBody(r.Body, &chat)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "unable to parse req body")
		return
	}
	chat.Sender = Sender
	err = dbHelpers.InsertMessage(chat)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	err = dbHelpers.SendNotification(chat)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), "failed to send message notification")
		return
	}
	chats, err := dbHelpers.GetAllMessage(chat.OrderID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error(), err.Error())
		return
	}
	utils.RespondJSON(w, 200, chats)
}
