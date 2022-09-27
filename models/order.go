package models

import (
	"github.com/lib/pq"
	"github.com/volatiletech/null"
	"time"
)

type OrderStatus string
type OrderType string
type OrderMode string
type TimeInterval int
type GeoDistance int

const (
	RadiusForSearch GeoDistance = 5
)
const (
	Processing           OrderStatus = "processing"
	Accepted             OrderStatus = "accepted"
	OutForDelivery       OrderStatus = "outForDelivery"
	Delivered            OrderStatus = "delivered"
	Cancelled            OrderStatus = "cancelled"
	Declined             OrderStatus = "declined"
	ScheduledOrderStatus OrderStatus = "scheduled"
)
const (
	TimeForStaffToAcceptOrder        TimeInterval = 30
	TimeForStoreManagerToAssignOrder TimeInterval = 180
	TimeToActivateScheduledOrder     TimeInterval = 900
)
const (
	CartMode     OrderMode = "cart"
	DeliveryMode OrderMode = "delivery"
)

const (
	Now       OrderType = "now"
	Scheduled OrderType = "scheduled"
)

type Order struct {
	ID            int          `json:"id" db:"id"`
	Mode          OrderMode    `json:"mode" db:"mode"`
	Type          OrderType    `json:"type" db:"order_type"`
	UserID        int          `json:"userID" db:"user_id"`
	StaffID       null.Int     `json:"staffID" db:"staff_id"`
	StaffName     null.String  `json:"staffName" db:"staff_name"`
	AddressID     int          `json:"addressID" db:"address_id"`
	Status        OrderStatus  `json:"status" db:"status"`
	OTP           null.Int     `json:"otp" db:"otp"`
	Amount        float32      `json:"amount" db:"amount"`
	Items         []ItemInfo   `json:"items" db:"-"`
	UserRating    null.Float32 `json:"-" db:"user_rating"`
	StaffRating   null.Float32 `json:"staffRating" db:"staff_rating"`
	DeliveryTime  string       `json:"deliveryTime" db:"delivery_time"`
	CreatedAt     string       `json:"createdAt" db:"created_at"`
	UpdatedAt     null.String  `json:"updatedAt" db:"updated_at"`
	Address       Address      `json:"address"`
	StoreMangerID int          `json:"-"`
}

type LocationStatus struct {
	OrderID   int          `json:"-" db:"order_id"`
	Status    OrderStatus  `json:"status" db:"status"`
	StaffID   null.Int     `json:"staffID" db:"staff_id"`
	Lat       null.Float64 `json:"lat" db:"lat"`
	Long      null.Float64 `json:"long" db:"long"`
	CreatedAt time.Time    `json:"-" db:"created_at"`
}

type Location struct {
	Id   int          `json:"id" db:"id"`
	Lat  null.Float64 `json:"lat" db:"lat"`
	Long null.Float64 `json:"long" db:"long"`
}

type ItemInfo struct {
	Id                 int          `json:"itemId" db:"item_id"`
	Quantity           int          `json:"quantity" db:"quantity"`
	Price              float32      `json:"price" db:"price"`
	StrikeThroughPrice null.Float32 `json:"strikeThroughPrice" db:"strikethrough_price"`
	Name               string       `json:"name" db:"name"`
	Category           string       `json:"category" db:"category"`
	BaseQuantity       string       `json:"baseQuantity" db:"base_quantity"`
	ImageID            int          `json:"-" db:"image_id"`
	ItemImageLinks     []string     `json:"itemImageLinks" db:"-"`
	Discount           null.Int     `json:"discount" db:"discount"`
	Bucket             null.String  `json:"-" db:"bucket"`
	Path               null.String  `json:"-" db:"path"`
}

type ScheduledOrder struct {
	ID         int      `json:"id" db:"id"`
	UserID     int      `json:"-" db:"user_id"`
	StaffID    null.Int `json:"staffId" db:"staff_id"`
	UserPhone  string   `json:"userPhone" db:"user_phone"`
	UserName   string   `json:"userName" db:"user_name"`
	StaffName  null.String   `json:"staffName" db:"staff_name"`
	StaffPhone null.String   `json:"staffPhone" db:"staff_phone"`
	AddressID     int            `json:"addressId" db:"address_id"`
	Mode          OrderMode      `json:"mode" db:"mode"`
	Weekdays      pq.StringArray `json:"weekdays" db:"weekdays"`
	Items         []ItemInfo     `json:"items" db:"-"`
	DeliveryTime  time.Time      `json:"deliveryTime" db:"delivery_time"`
	CreatedAt     time.Time      `json:"-" db:"created_at"`
	Amount        null.Float32   `json:"amount" db:"amount"`
	StartDate     time.Time      `json:"startDate" db:"start_date"`
	EndDate       *time.Time     `json:"endDate" db:"end_date"`
	Address       Address        `json:"address"`
	StoreMangerID int            `json:"-"`
}

type VerifyOrder struct {
	OTP        int          `json:"otp" db:"otp"`
	UserRating null.Float32 `json:"userRating" db:"user_rating"`
	Amount     float32      `json:"amount" db:"amount"`
}
type ConfirmOrder struct {
	StaffRating null.Float32 `json:"staffRating" db:"staff_rating"`
}

type OrderResponse struct {
	OrderID int       `db:"id"`
	Mode    OrderMode `db:"mode"`
	UserID  int       `db:"user_id"`
}

type DisputedOrder struct {
	OrderID     int         `json:"orderId" db:"order_id"`
	UserPhone   string      `json:"userPhone" db:"user_phone"`
	UserAddress string      `json:"userAddress" db:"address_data"`
	DisputedAt  string      `json:"disputedAt" db:"disputed_at"`
	ResolvedAt  null.String `json:"resolvedAt" db:"resolved_at"`
}

type DisputedOrderInfo struct {
	OrderID    int        `json:"orderId" db:"order_id"`
	OrderMode  OrderMode  `json:"orderMode" db:"mode"`
	UserName   string     `json:"userName" db:"user_name"`
	UserPhone  string     `json:"userPhone" db:"user_phone"`
	StaffName  string     `json:"staffName" db:"staff_name"`
	StaffPhone string     `json:"staffPhone" db:"staff_phone"`
	Amount     float32    `json:"amount"`
	Items      []ItemInfo `json:"items"`
}

type DeniedUnassignedOrders struct {
	OrderID      int       `json:"orderId" db:"order_id"`
	OrderMode    OrderMode `json:"orderMode" db:"mode"`
	OrderType    string    `json:"orderType" db:"order_type"`
	Status       string    `json:"status" db:"status"`
	AddressData  string    `json:"addressData" db:"address_data"`
	UserPhone    string    `json:"userPhone" db:"phone"`
	DeliveryTime time.Time `json:"deliveryTime" db:"delivery_time"`
}

type Response struct {
	Success bool `json:"success"`
}
