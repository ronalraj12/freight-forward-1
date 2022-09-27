package cronJobs

import (
	"database/sql"
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/dbHelpers"
	"github.com/RemoteState/yourdaily-server/firebase"
	"github.com/RemoteState/yourdaily-server/handlers"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/sirupsen/logrus"
	"time"
)

func CronFuncToCheckOrderStatus() {
	SQl := `
		UPDATE orders
		SET status = $1
		WHERE orders.id = ANY (SELECT id AS order_id
							   FROM orders
							   WHERE staff_id IS NULL
								 AND status = $2
								 AND (NOW() - delivery_time) >=  ($3 ||' second')::INTERVAL)
		RETURNING id AS order_id,user_id;
`
	userIDs := make([]struct {
		OrderID int   `json:"orderId" db:"order_id"`
		UserID  int64 `json:"userId" db:"user_id"`
	}, 0)
	err := database.YourDailyDB.Select(&userIDs, SQl, models.Declined, models.Processing, models.TimeForStoreManagerToAssignOrder)
	if err != nil {
		logrus.Errorf("CronFuncToCheckOrderStatus: error :%v", err)
		return
	}
	for _, v := range userIDs {
		go firebase.OrderStatusUpdateNotification(v.UserID, v.OrderID, models.Declined, "")
	}
}

// MoveScheduledOrders moves scheduled order to orders table if today's date = order's delivery day
func MoveScheduledOrders() {
	if time.Now().Hour() == 1 {
		err := dbHelpers.MoveScheduledOrders()
		if err != nil {
			logrus.Errorf("failed to move scheduled orders to now with error::%v", err)
			return
		} else {
			logrus.Println("scheduled orders moved to now successfully!")
		}
	}
}

func InitiateScheduledOrder() {
	query := `
		SELECT id,address_id,user_id,mode
		FROM orders
		WHERE status = 'scheduled'
		  AND orders.delivery_time - NOW() < (900 || 'second')::INTERVAL
`
	orders := make([]struct {
		OrderID   int              `db:"id"`
		UserID    int              `db:"user_id"`
		AddressId int              `db:"address_id"`
		Mode      models.OrderMode `db:"mode"`
	}, 0)
	err := database.YourDailyDB.Select(&orders, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return
		} else {
			logrus.Errorf("failed to retrive any orders")
		}
	}
	status := models.Processing
	for _, order := range orders {
		go handlers.FindAndPing(order.Mode, order.AddressId, order.UserID, order.OrderID)

		err := dbHelpers.UpdateOrders(order.OrderID, nil, nil, nil, nil, &status, &order.UserID)
		if err != nil {
			logrus.Error(err)
		}
	}

}
