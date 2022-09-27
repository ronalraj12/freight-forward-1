package dbHelpers

import (
	"database/sql"
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null"
	"time"
)

// InsertItem creates a new item entry in table

func InsertItem(name string, price float32, inStock bool, categoryID int, baseQuantity string, strikeThroughPrice null.Float32) (int, error) {
	SQL := `INSERT INTO items(name, price, in_stock, category, base_quantity,strikethrough_price) VALUES ($1, $2, $3, $4,$5, null) RETURNING id`
	if strikeThroughPrice.Valid {
		SQL = `INSERT INTO items(name, price, in_stock, category, base_quantity,strikethrough_price) VALUES ($1, $2, $3, $4, $5,$6) RETURNING id`
		var itemID int
		err := database.YourDailyDB.Get(&itemID, SQL, name, price, inStock, categoryID, baseQuantity, strikeThroughPrice)
		return itemID, err
	}
	var itemID int
	err := database.YourDailyDB.Get(&itemID, SQL, name, price, inStock, categoryID, baseQuantity)
	return itemID, err

}

// GetItems returns all items
func GetItems(OutOfStock bool) ([]models.Item, error) {
	SQL := `SELECT
			id,
			name,
			price,
			in_stock,
			created_at,
			category,
       		base_quantity,
     		strikethrough_price
		FROM items
		WHERE archived_at IS NULL 
`
	if !OutOfStock {
		SQL += `AND in_stock = TRUE `
	}
	SQL += `ORDER BY created_at DESC `
	items := make([]models.Item, 0)

	err := database.YourDailyDB.Select(&items, SQL)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func GetItemCount() (int, error) {
	SQL := `SELECT count(*) 
		FROM items
		WHERE archived_at IS NULL`
	var items int
	err := database.YourDailyDB.Get(&items, SQL)
	return items, err
}

// ModifyItem modifies a given item in table
func ModifyItem(name string, price float32, inStock bool, categoryID, itemID int, baseQuantity string, strikeThroughPrice float32) error {
	SQL := `UPDATE items
			SET name          = $1,
				price         = $2,
				in_stock      = $3,
				updated_at    = $4,
				category      = $5,
				base_quantity = $7,
			    strikethrough_price = $8
			WHERE id = $6`
	_, err := database.YourDailyDB.Exec(SQL, name, price, inStock, time.Now(), categoryID, itemID, baseQuantity, strikeThroughPrice)
	return err
}

// GetItemById gets the item details for a given id
func GetItemById(itemID int) (*models.Item, error) {
	SQL := `SELECT
			id,
			name,
			price,
			in_stock,
			created_at,
			category,
       		base_quantity,
     		strikethrough_price
		FROM items
		WHERE archived_at IS NULL
		AND id = $1`

	var item models.Item

	err := database.YourDailyDB.Get(&item, SQL, itemID)

	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ArchiveItem archives a given item
func ArchiveItem(itemID int) error {
	SQL := `UPDATE items 
			SET archived_at  = $1 
			WHERE archived_at IS NULL 
			AND id = $2`
	result, err := database.YourDailyDB.Exec(SQL, time.Now(), itemID)
	if err != nil {
		return err
	}
	affectedCount, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affectedCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// LinkItemWithImage stores relation between an image and item
func LinkItemWithImage(itemID, imageID int) error {
	err := database.Tx(func(tx *sqlx.Tx) error {
		SQL := `DELETE FROM item_images where item_id = $1`
		_, err := tx.Exec(SQL, itemID)
		if err != nil {
			return err
		}

		SQL = `INSERT INTO item_images(item_id, image_id) VALUES ($1, $2)`
		_, err = tx.Exec(SQL, itemID, imageID)
		return err
	})
	return err
}

// GetImageInfoByItemID returns image info - bucket & path for given itemID
func GetImageInfoByItemID(itemID int) ([]models.Image, error) {
	SQL := `SELECT i.bucket, i.path
		FROM images i
		LEFT JOIN item_images ii
		ON i.id = ii.image_id
		WHERE ii.item_id = $1
		ORDER BY i.created_at DESC`

	imagesInfo := make([]models.Image, 0)
	err := database.YourDailyDB.Select(&imagesInfo, SQL, itemID)
	if err != nil {
		if err == sql.ErrNoRows {
			return imagesInfo, nil
		}
		return nil, err
	}
	return imagesInfo, nil
}
