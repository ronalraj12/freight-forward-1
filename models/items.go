package models

import (
	"github.com/volatiletech/null"
	"time"
)

type Item struct {
	ID                 int          `json:"id" db:"id"`
	CategoryID         int          `json:"categoryID" db:"category"`
	Name               string       `json:"name" db:"name"`
	Price              float32      `json:"price" db:"price"`
	StrikeThroughPrice null.Float32 `json:"strikeThroughPrice" db:"strikethrough_price"`
	InStock            bool         `json:"inStock" db:"in_stock"`
	CreatedAt          time.Time    `json:"-" db:"created_at"`
	ItemImageLinks     []string     `json:"itemImageLinks" db:"-"`
	BaseQuantity       string       `json:"baseQuantity" db:"base_quantity"`
}

type ItemCategory struct {
	ID        int       `json:"id" db:"id"`
	Category  string    `json:"category" db:"category"`
	CreatedAt time.Time `json:"-" db:"created_at"`
}
