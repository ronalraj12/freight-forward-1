package handlers

import (
	"database/sql"
	"github.com/RemoteState/yourdaily-server/dbHelpers"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"github.com/go-chi/chi"
	"net/http"
)

func CreateCategory(w http.ResponseWriter, r *http.Request) {

	reqBody := struct {
		Category string `json:"category"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}

	categoryID, err := dbHelpers.InsertCategory(reqBody.Category)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to store category entry")
		return
	}

	category, err := dbHelpers.GetCategoryById(categoryID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get category")
		return
	}
	utils.RespondJSON(w, http.StatusCreated, category)
}

func GetAllCategories(w http.ResponseWriter, r *http.Request) {

	categories, err := dbHelpers.GetCategories()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get category entries")
		return
	}
	utils.RespondJSON(w, http.StatusOK, categories)
}

func ModifyCategory(w http.ResponseWriter, r *http.Request) {

	categoryID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to convert given categoryID to int")
		return
	}

	reqBody := struct {
		Category string `json:"category"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}
	if err := dbHelpers.ModifyCategory(reqBody.Category, categoryID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update category entry")
		return
	}

	category, err := dbHelpers.GetCategoryById(categoryID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get category")
		return
	}
	utils.RespondJSON(w, http.StatusCreated, category)
}

func ArchiveCategory(w http.ResponseWriter, r *http.Request) {

	categoryID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to convert given categoryID to int")
		return
	}

	if err = dbHelpers.ArchiveCategory(categoryID); err != nil {
		if err == sql.ErrNoRows {
			utils.RespondError(w, http.StatusBadRequest, err, "Failed to archive given category")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to archive given category")
		return
	}
	utils.RespondJSON(w,200,models.Response{
		Success: true,
	})
}
