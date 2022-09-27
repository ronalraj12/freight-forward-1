package handlers

import (
	"database/sql"
	"github.com/RemoteState/yourdaily-server/dbHelpers"
	"github.com/RemoteState/yourdaily-server/firebase"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"github.com/go-chi/chi"
	"github.com/volatiletech/null"
	"net/http"
)

func CreateItem(w http.ResponseWriter, r *http.Request) {

	reqBody := struct {
		CategoryID         int          `json:"category"`
		Name               string       `json:"name"`
		Price              float32      `json:"price"`
		StrikeThroughPrice null.Float32 `json:"strikeThroughPrice"`
		InStock            bool         `json:"inStock"`
		BaseQuantity       string       `json:"baseQuantity"`
		ImageID            int          `json:"imageId"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}
	itemID, err := dbHelpers.InsertItem(reqBody.Name, reqBody.Price, reqBody.InStock, reqBody.CategoryID, reqBody.BaseQuantity, reqBody.StrikeThroughPrice)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to store item entry")
		return
	}

	if err := dbHelpers.LinkItemWithImage(itemID, reqBody.ImageID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in storing item image relation")
		return
	}

	item, err := dbHelpers.GetItemById(itemID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get item")
		return
	}

	imagesInfo, err := dbHelpers.GetImageInfoByItemID(itemID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in generating image link")
		return
	}

	for i := range imagesInfo {
		imageLink, err := firebase.GetURL(&imagesInfo[i])
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed in getting image URL")
			return
		}
		item.ItemImageLinks = append(item.ItemImageLinks, imageLink)
	}

	utils.RespondJSON(w, http.StatusCreated, item)
}

func GetAllItems(w http.ResponseWriter, r *http.Request) {

	items, err := dbHelpers.GetItems(false)
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

func ModifyItem(w http.ResponseWriter, r *http.Request) {

	itemID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to convert given itemID to int")
		return
	}
	reqBody := struct {
		CategoryID         int     `json:"category"`
		Name               string  `json:"name"`
		Price              float32 `json:"price"`
		InStock            bool    `json:"inStock"`
		BaseQuantity       string  `json:"baseQuantity"`
		StrikeThroughPrice float32 `json:"strikeThroughPrice"`
	}{}

	if err := utils.ParseBody(r.Body, &reqBody); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "Failed to decode request body")
		return
	}
	if err := dbHelpers.ModifyItem(reqBody.Name, reqBody.Price, reqBody.InStock, reqBody.CategoryID, itemID, reqBody.BaseQuantity, reqBody.StrikeThroughPrice); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update item entry")
		return
	}

	item, err := dbHelpers.GetItemById(itemID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get item")
		return
	}
	utils.RespondJSON(w, http.StatusCreated, item)
}

func ArchiveItem(w http.ResponseWriter, r *http.Request) {

	itemID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to convert given itemID to int")
		return
	}

	if err = dbHelpers.ArchiveItem(itemID); err != nil {
		if err == sql.ErrNoRows {
			utils.RespondError(w, http.StatusBadRequest, err, "Failed to archive given item")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to archive given item")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func AddImageForExistingItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error(), "Invalid item ID")
		return
	}

	file, fileBytes, downloadedFileName, err := utils.ReadFromFile(r, string(models.ItemImage))
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

	imageID, err := dbHelpers.StoreImageInfo(models.BucketLink, uploadedFileName, string(models.ItemImage))
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in storing uploaded file info")
		return
	}

	if err := dbHelpers.LinkItemWithImage(itemID, imageID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in storing item image relation")
		return
	}

	imagesInfo, err := dbHelpers.GetImageInfoByItemID(itemID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in generating image link")
		return
	}

	var urls []string
	for i := range imagesInfo {
		url, err := firebase.GetURL(&imagesInfo[i])
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed in getting image URL")
			return
		}
		urls = append(urls, url)
	}

	response := struct {
		ImageURL []string `json:"imageURL"`
	}{
		ImageURL: urls,
	}
	utils.RespondJSON(w, http.StatusOK, response)
}

func GetItemById(w http.ResponseWriter, r *http.Request) {

	itemID, err := utils.StringToInt(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to convert given itemID to int")
		return
	}

	item, err := dbHelpers.GetItemById(itemID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get item")
		return
	}

	imagesInfo, err := dbHelpers.GetImageInfoByItemID(itemID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed in generating image link")
		return
	}

	for i := range imagesInfo {
		url, err := firebase.GetURL(&imagesInfo[i])
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed in getting image URL")
			return
		}
		item.ItemImageLinks = append(item.ItemImageLinks, url)
	}

	utils.RespondJSON(w, http.StatusOK, item)
}
