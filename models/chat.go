package models

type Chat struct {
	OrderID   int    `json:"orderID" db:"order_id"`
	Sender    int    `json:"sender" db:"sender"`
	Message   string `json:"message" db:"message"`
	CreatedAt string `json:"createdAt" db:"created_at"`
}
