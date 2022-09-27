package firebase

import (
	"context"
	"firebase.google.com/go/messaging"
	"fmt"
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"time"
)

type MessageType string

const (
	MessageTypeNewOrderNotification    = "NewOrderNotification"
	MessageTypeOrderStatusUpdate       = "OrderStatusUpdateNotification"
	MessageTypeChatNotification        = "ChatNotification"
	MessageTypeScheduledOrderCancelled = "ScheduledOrderCancelled"
)

func SendNewOrderNotificationToStaff(userIds []int64, orderId int, lat, long float64, addressData string) error {
	logrus.Infof("sending notification to %+v", userIds)

	SQL := `SELECT token
				FROM fcm_token
				WHERE user_id = ANY ($1)`
	var registrationTokens []string
	err := database.YourDailyDB.Select(&registrationTokens, SQL, pq.Int64Array(userIds))
	if err != nil {
		return err
	}

	message := &messaging.MulticastMessage{
		Data: map[string]string{
			"type":        MessageTypeNewOrderNotification,
			"orderId":     fmt.Sprintf("%d", orderId),
			"lat":         fmt.Sprintf("%f", lat),
			"long":        fmt.Sprintf("%f", long),
			"addressData": addressData,
			"expireTime":  time.Now().Add(30 * time.Second).Format(time.RFC3339Nano),
		},
		Tokens: registrationTokens,
	}

	resp, err := FirebaseClient.SendMulticast(context.Background(), message)
	if err != nil {
		logrus.Errorf("SendNewOrderNotificationToStaff: Error while sending push notifications %v", err)
		return err
	}
	fmt.Print(resp)
	logrus.Infof("notification to %+v succesffuly", userIds)

	return nil
}

func OrderStatusUpdateNotification(userId int64, orderId int, status models.OrderStatus, addressData string) {
	logrus.Infof("sending notification to %+v", userId)

	// language=SQL
	SQL := `
	SELECT token
	FROM fcm_token
	WHERE user_id = $1
`
	var registrationToken string
	database.YourDailyDB.Get(&registrationToken, SQL, userId)
	if registrationToken == "" {
		logrus.Errorf("no token found for userID  %d orderid = %d", userId, orderId)
		return
	}
	content := ""
	switch status {
	case models.Accepted:
		content = "Order Accepted"
	case models.OutForDelivery:
		content = "Order is out for delivery"
	case models.Cancelled:
		content = "order is cancelled"
	case models.Delivered:
		content = "order is delivered Successfully"
	case models.Declined:
		content = "order has been declined,please try again!"
	default:
		content = "order status has been updated"
	}
	message := &messaging.MulticastMessage{
		Data: map[string]string{
			"type":    MessageTypeOrderStatusUpdate,
			"title":   "Order Status Update",
			"status":  string(status),
			"content": content,
			"address": addressData,
			"orderId": fmt.Sprintf("%d", orderId),
		},
		Tokens: []string{registrationToken},
	}

	_, err := FirebaseClient.SendMulticast(context.Background(), message)
	if err != nil {
		logrus.Errorf("OrderStatusUpdateNotification: Error while sending push notifications message %+v and error %v", message, err)
		return
	}
	logrus.Infof("notification succesfull to user %d with message %+v", userId, message)
}

func NewMessageNotification(userID, orderID int, message string) {
	logrus.Infof("sending chat notification to %+v", userID)

	// language=SQL
	SQL := `
	SELECT token
	FROM fcm_token
	WHERE user_id = $1
`
	var registrationToken string
	database.YourDailyDB.Get(&registrationToken, SQL, userID)
	if registrationToken == "" {
		logrus.Errorf("no token found for userID  %d", userID)
		return
	}

	payLoad := &messaging.MulticastMessage{
		Data: map[string]string{
			"type":    MessageTypeChatNotification,
			"title":   "New message",
			"message": message,
			"orderId": fmt.Sprintf("%d", orderID)},
		Tokens: []string{registrationToken},
	}

	_, err := FirebaseClient.SendMulticast(context.Background(), payLoad)
	if err != nil {
		logrus.Errorf("NewMessageNotification: Error while sending push notifications message %+v and error %v", message, err)
		return
	}
	logrus.Infof("notification chat message succesfull to user %d with message %+v", userID, message)
}

func ScheduledOrderCanceledNotification(userID, orderID int, message string) {
	logrus.Infof("sending chat notification to %+v", userID)

	// language=SQL
	SQL := `
	SELECT token
	FROM fcm_token
	WHERE user_id = $1
`
	var registrationToken string
	database.YourDailyDB.Get(&registrationToken, SQL, userID)
	if registrationToken == "" {
		logrus.Errorf("no token found for userID  %d", userID)
		return
	}

	payLoad := &messaging.MulticastMessage{
		Data: map[string]string{
			"type":    MessageTypeScheduledOrderCancelled,
			"title":   "Scheduled Order Canceled",
			"message": message,
			"orderId": fmt.Sprintf("%d", orderID)},
		Tokens: []string{registrationToken},
	}

	_, err := FirebaseClient.SendMulticast(context.Background(), payLoad)
	if err != nil {
		logrus.Errorf("NewMessageNotification: Error while sending push notifications message %+v and error %v", message, err)
		return
	}
	logrus.Infof("notification chat message succesfull to user %d with message %+v", userID, message)
}
