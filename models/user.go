package models

import (
	"github.com/volatiletech/null"
	"time"
)

type UserPermission string

const (
	DefaultUser  UserPermission = "user"
	CartBoy      UserPermission = "cart-boy"
	DeliveryBoy  UserPermission = "delivery-boy"
	StoreManager UserPermission = "store-manager"
	Admin        UserPermission = "admin"
	Guest        UserPermission = "guest"
)

const (
	MaxFlagCount int = 50
)

type User struct {
	ID               int              `json:"id" db:"id"`
	Name             null.String      `json:"name" db:"name"`
	Phone            string           `json:"phone" db:"phone"`
	Email            null.String      `json:"email" db:"email"`
	Rating           float32          `json:"rating" db:"rating"`
	ProfileImageID   null.Int         `json:"-" db:"profile_image"`
	ProfileImageLink string           `json:"profileImageLink" db:"-"`
	CreatedAt        time.Time        `json:"-" db:"created_at"`
	Permissions      []UserPermission `json:"permissions" db:"-"`
	Password         string           `json:"-" db:"password"`
	Permission       UserPermission   `json:"-" db:"permission_type"`
	AllowedMode      OrderMode        `json:"-" db:"-"`
}
