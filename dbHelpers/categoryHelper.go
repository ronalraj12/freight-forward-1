package dbHelpers

import (
	"database/sql"
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/models"
	"time"
)

// InsertCategory creates a new category entry in table
func InsertCategory(category string) (int, error) {
	SQL := `INSERT INTO categories(category) VALUES ($1) RETURNING id`
	var categoryID int
	err := database.YourDailyDB.Get(&categoryID, SQL, category)
	return categoryID, err
}

// GetCategories returns all categories
func GetCategories() ([]models.ItemCategory, error) {
	SQL := `SELECT 
				id,
       			category,
       			created_at
			FROM categories 
			WHERE  archived_at IS NULL`

	categories := make([]models.ItemCategory, 0)

	err := database.YourDailyDB.Select(&categories, SQL)
	if err != nil {
		return nil, err
	}
	return categories, nil
}

// ModifyCategory modifies a given category in table
func ModifyCategory(category string, categoryID int) error {
	SQL := `UPDATE categories SET category = $1, updated_at = $2 WHERE id = $3`
	_, err := database.YourDailyDB.Exec(SQL, category, time.Now(), categoryID)
	return err
}

// GetCategoryById gets the category details for a given id
func GetCategoryById(categoryID int) (*models.ItemCategory, error) {
	SQL := `SELECT 
				id,
       			category,
       			created_at
			FROM categories 
			WHERE  archived_at IS NULL
			AND id = $1`

	var category models.ItemCategory

	err := database.YourDailyDB.Get(&category, SQL, categoryID)
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// ArchiveCategory archives a given category
// need to make sure that no item is assigned to this category
func ArchiveCategory(categoryID int) error {
	SQL := `UPDATE categories
			SET archived_at = $1
			WHERE archived_at IS NULL
			AND id = $2
			AND NOT EXISTS (select 1 from items WHERE items.category = categories.id and items.archived_at is not null);`
	result, err := database.YourDailyDB.Exec(SQL, time.Now(), categoryID)
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
