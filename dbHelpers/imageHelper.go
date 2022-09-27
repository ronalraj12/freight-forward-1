package dbHelpers

import (
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/firebase"
	"github.com/RemoteState/yourdaily-server/models"
)

//GetImageUrl take image id and return image url
func GetImageUrl(imageId int) (string, error) {
	SQL := `SELECT bucket, path
		FROM images where id = $1`

	imagesInfo := models.Image{}
	err := database.YourDailyDB.Get(&imagesInfo, SQL, imageId)
	if err != nil {
		return "", err
	}
	url, err := firebase.GetURL(&imagesInfo)
	if err != nil {
		return "", err
	}
	return url, nil
}
