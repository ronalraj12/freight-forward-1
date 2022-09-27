package models

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/lib/pq"
	"github.com/volatiletech/null"
	"time"
)

type DashBoardStats struct {
	UserCount          int `json:"userCount" db:"user_count"`
	CartBoyCount       int `json:"cartBoyCount" db:"cart_boy_count"`
	DeliveryBoyCount   int `json:"deliveryBoyCount" db:"delivery_boy_count"`
	TotalItems         int `json:"totalItems"`
	UnassignedOrders   int `json:"unassignedOrders"`
	OnGoingOrder       int `json:"onGoingOrder"`
	DeniedOrder        int `json:"deniedOrder"`
	DisputedOrder      int `json:"disputedOrder"`
	ScheduledOrder     int `json:"scheduledOrder"`
	ActiveUsers        int `json:"activeUsers"`
	UnapprovedStaff    int `json:"unapprovedStaff"`
	BookingForLastWeek int `json:"bookingForLastWeek"`
}

type OrderNSStats struct {
	Date            string `json:"date" db:"order_date"`
	ScheduledOrders int    `json:"scheduledOrders" db:"scheduled_orders"`
	NowOrders       int    `json:"nowOrders" db:"now_orders"`
}

type OrderADStats struct {
	Date           string `json:"date" db:"order_date"`
	AcceptedOrders int    `json:"acceptedOrders" db:"accepted_orders"`
	DeclinedOrders int    `json:"declinedOrders" db:"declined_orders"`
}

type DashBoardStaffDetails struct {
	ID               int       `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Contact          string    `json:"contact" db:"phone"`
	RegDate          time.Time `json:"regDate" db:"created_at"`
	TotalOrders      int       `json:"totalOrders" db:"total_order"`
	DeniedOrders     int       `json:"deniedOrders" db:"denied_orders"`
	CanceledOrders   int       `json:"canceledOrders" db:"cancelled_orders"`
	TotalAmount      float32   `json:"totalAmount" db:"total_amount"`
	AvgRating        float32   `json:"avgRating" db:"avg_rating"`
	Flagged          int       `json:"flagged" db:"flagged"`
	Enabled          bool      `json:"enabled" db:"enabled"`
	ProfileImageLink string    `json:"profileImageLink" db:"-"`
}

type DashBoardUserDetails struct {
	ID               int            `json:"id" db:"id"`
	Name             string         `json:"name" db:"name"`
	Contact          string         `json:"contact" db:"phone"`
	RegDate          time.Time      `json:"regDate" db:"created_at"`
	DefaultAddress   null.String    `json:"defaultAddress" db:"default_address"`
	DefaultLat       null.Float64   `json:"defaultAddressLat" db:"default_lat"`
	DefaultLong      null.Float64   `json:"defaultAddressLong" db:"default_long"`
	TotalOrders      int            `json:"totalOrders" db:"total_order"`
	TotalAmount      float32        `json:"totalAmount" db:"total_amount"`
	DeniedOrders     int            `json:"deniedOrders" db:"denied_orders"`
	CanceledOrders   int            `json:"canceledOrders" db:"canceled_orders"`
	AvgRating        float32        `json:"avgRating" db:"avg_rating"`
	FlagCount        int            `json:"flagCount" db:"flag_count"`
	ProfileImageLink string         `json:"profileImageLink" db:"-"`
	TopThreeLocation []Address      `json:"topThreeLocation"`
	TopThreeItems    pq.StringArray `json:"topThreeItems" db:"top_three_items"`
}

type OrderInfo struct {
	OrderID   int       `json:"orderId" db:"-"`
	Mode      OrderMode `json:"mode" db:"mode"`
	UserID    int       `json:"userId" db:"user_id"`
	AddressID int       `json:"AddressId" db:"address_id"`
}

type StoreMangerCred struct {
	ID         int            `json:"-" db:"id"`
	Name       string         `json:"name" db:"name"`
	Email      string         `json:"email" db:"email"`
	Password   string         `json:"password" db:"password"`
	Permission UserPermission `json:"-" db:"permission"`
}

type JWTClaims struct {
	UserID int    `json:"userId"`
	Email  string `json:"email"`
	jwt.StandardClaims
}

type Offer struct {
	ID          int         `json:"id" db:"id"`
	Title       null.String `json:"title" db:"title"`
	Description null.String `json:"description" db:"description"`
	Discount    int         `json:"discount" db:"discount"`
	ImageID     null.Int    `json:"-" db:"image_id"`
	ImageURL    string      `json:"imageUrl" db:"-"`
}

type ScheduledOrderCsv struct {
	OrderID      int         `json:"orderID" db:"order_id"`
	OrderType    string      `json:"orderType" db:"order_type"`
	OrderMode    string      `json:"orderMode" db:"order_mode"`
	Status       string      `json:"status" db:"status"`
	UserName     string      `json:"userName" db:"user_name"`
	UserPhone    string      `json:"userPhone" db:"user_phone"`
	UserAddress  string      `json:"userAddress" db:"user_address"`
	DeliveryTime time.Time   `json:"deliveryTime" db:"delivery_time"`
	StaffName    null.String `json:"staffName" db:"staff_name"`
	StaffPhone   null.String `json:"staffPhone" db:"staff_phone"`
	CreatedAt    time.Time   `json:"createdAt" db:"created_at"`
	StartDate    time.Time   `json:"startDate" db:"start_date"`
	EndDate      time.Time   `json:"endDate" db:"end_date"`
	Items        []ItemInfo  `json:"items" db:"items"`
}
