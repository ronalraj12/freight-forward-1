package models

const DefaultAddressTag = "home"

type Address struct {
	ID          int     `json:"id" db:"id"`
	UserID      int     `json:"userId" db:"user_id"`
	AddressData string  `json:"addressData" db:"address_data"`
	Lat         float64 `json:"latitude" db:"lat"`
	Long        float64 `json:"longitude" db:"long"`
	AddressTag  string  `json:"addressTag" db:"address_tag"`
	IsDefault   bool    `json:"isDefaultAddress" db:"is_default"`
	CreatedAt   string  `json:"createdAt" db:"created_at"`
	UpdatedAt   string  `json:"updatedAt" db:"updated_at"`
	ArchivedAt  string  `json:"-" db:"archived_at"`
}

type GeoLocation struct {
	Lat  float64 `json:"lat" db:"lat"`
	Long float64 `json:"long" db:"long"`
}
