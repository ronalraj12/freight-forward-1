package dbHelpers

import (
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/firebase"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/volatiletech/null"
)

func InsertMessage(chat models.Chat) error {
	query := `Insert into chat (order_id, sender, message, created_at)
				values ($1,$2,$3,now())`

	_, err := database.YourDailyDB.Exec(query, chat.OrderID, chat.Sender, chat.Message)
	return err
}

func GetAllMessage(orderID int) ([]models.Chat, error) {
	query := `select order_id, sender, message, created_at
			from chat
			where order_id = $1 order by created_at`
	chats := make([]models.Chat, 0)
	err := database.YourDailyDB.Select(&chats, query, orderID)
	return chats, err
}

func SendNotification(chat models.Chat) error {
	query := `
		SELECT user_id, staff_id
		FROM orders
		WHERE id = $1
`
	var userId, staffId null.Int
	err := database.YourDailyDB.QueryRowx(query, chat.OrderID).Scan(&userId, &staffId)
	if err != nil {
		return err
	}
	var isStaff bool
	if staffId.Valid {
		isStaff = staffId.Int == chat.Sender
	}
	if isStaff {
		go firebase.NewMessageNotification(userId.Int, chat.OrderID, chat.Message)
	} else {
		if staffId.Valid {
			go firebase.NewMessageNotification(staffId.Int, chat.OrderID, chat.Message)
		}
	}
	return nil
}
