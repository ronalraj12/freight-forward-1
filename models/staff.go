package models

import (
	"github.com/volatiletech/null"
	"time"
)

const OrderAcceptanceLimit int = 3

type StaffOrder struct {
	OrderId         int          `json:"orderID" db:"order_id"`
	UserID          int          `json:"-" db:"user_id"`
	UserName        string       `json:"userName" db:"name"`
	UserPhone       string       `json:"userPhone" db:"phone"`
	UserImage       string       `json:"user_image" db:"phone"`
	OrderType       string       `json:"orderType" db:"order_type"`
	DeliveryTime    null.Time    `json:"deliveryTime" db:"delivery_time"`
	StaffRating     null.Float32 `json:"staffRating" db:"staff_rating"`
	UserRating      null.Float32 `json:"userRating" db:"user_rating"`
	ItemBytes       []byte       `json:"-" db:"item_byte"`
	Items           []ItemInfo   `json:"items" db:"-"`
	Amount          float32      `json:"amount" db:"amount"`
	UserAddressData string       `json:"userAddressData" db:"address_data"`
	UserLat         null.Float64 `json:"userLat" db:"lat"`
	UserLong        null.Float64 `json:"userLong" db:"long"`
	Status          OrderStatus  `json:"status" db:"status"`
}

type UnapprovedStaff struct {
	ID        int         `json:"id" db:"id"`
	Name      null.String `json:"name" db:"name"`
	Phone     string      `json:"phone" db:"phone"`
	CreatedAt time.Time   `json:"createdAt" db:"created_at"`
}

type OrderNotification struct {
	OrderID     int     `json:"orderId" db:"order_id"`
	AddressData string  `json:"addressData" db:"address_data"`
	Lat         float64 `json:"lat" db:"lat"`
	Long        float64 `json:"long" db:"long"`
	ExpireTime  int64   `json:"expireTime" db:"expire_time"`
}

type OrderAccept struct {
	OrderID  int    `json:"orderId"`
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}
