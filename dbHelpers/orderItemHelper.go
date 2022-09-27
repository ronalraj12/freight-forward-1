package dbHelpers

import (
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/firebase"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/sirupsen/logrus"
)

func GetOrderItems(orderId int) ([]models.ItemInfo, error) {
	SQL := `SELECT
				name,
				category,
				price,
				base_quantity,
				quantity,
       			discount,
       			bucket,
       			path,
       			strikethrough_price
			 FROM order_items
			 WHERE order_id = $1`

	items := make([]models.ItemInfo, 0)
	err := database.YourDailyDB.Select(&items, SQL, orderId)
	for i := range items {
		if items[i].Bucket.Valid {
			imageInfo := models.Image{
				Bucket: items[i].Bucket.String,
				Path:   items[i].Path.String,
			}
			imageLink, err := firebase.GetURL(&imageInfo)
			if err != nil {
				logrus.Errorf("failed to fetch image url item id: %d  ", items[i].Id)
			} else {
				items[i].ItemImageLinks = append(items[i].ItemImageLinks, imageLink)
			}
		}
	}
	return items, err
}

func GetScheduledOrderItems(orderId int) ([]models.ItemInfo, error) {
	SQL := `SELECT
				name,
				category,
				base_quantity,
				quantity,
       			bucket,
       			path
			 FROM scheduled_ordered_items
			 WHERE order_id = $1`

	items := make([]models.ItemInfo, 0)
	err := database.YourDailyDB.Select(&items, SQL, orderId)
	for i := range items {
		if items[i].Bucket.Valid {
			imageInfo := models.Image{
				Bucket: items[i].Bucket.String,
				Path:   items[i].Path.String,
			}
			imageLink, err := firebase.GetURL(&imageInfo)
			if err != nil {
				logrus.Errorf("failed to fetch image url item id: %d  ", items[i].Id)
			} else {
				items[i].ItemImageLinks = append(items[i].ItemImageLinks, imageLink)
			}
		}
	}
	return items, err
}
