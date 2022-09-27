package dbHelpers

import (
	"database/sql"
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/jmoiron/sqlx"
	"time"
)

//InsertAddress insert data from the address models into the address table
func InsertAddress(data models.Address) error {

	err := database.Tx(func(tx *sqlx.Tx) error {
		if data.IsDefault {
			query := `UPDATE address SET is_default = $1 WHERE address.user_id = $2`
			_, err := tx.Exec(query, false, data.UserID)
			if err != nil {
				return err
			}
		}
		query := "INSERT INTO address (user_id,address_data,address_tag,lat,long,is_default,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)"

		_, err := tx.Exec(query, data.UserID, data.AddressData, data.AddressTag, data.Lat, data.Long, data.IsDefault, time.Now().Format(time.RFC3339Nano))
		return err
	})
	return err

}

//SelectAllAddress return all the address for the received userID and return as a slice of address models
func SelectAllAddress(userID int) ([]models.Address, error) {

	query := `SELECT id,user_id,address_tag,address_data,lat,long,created_at,is_default,updated_at 
				FROM address 
				WHERE user_id=$1 and archived_at is null order by created_at desc`

	addresses := make([]models.Address, 0)

	err := database.YourDailyDB.Select(&addresses, query, userID)
	return addresses, err
}

//UpdateAddress updates the data for given addID for the userID if no update is done return error of NoRowsAffected
func UpdateAddress(data models.Address) error {

	err := database.Tx(func(tx *sqlx.Tx) error {
		if data.IsDefault {
			query := `UPDATE address SET is_default = $1 WHERE address.user_id = $2`
			_, err := tx.Exec(query, false, data.UserID)

			if err != nil {
				return err
			}
		}

		query := `UPDATE address 
				SET updated_at=now(),is_default=$1 
				WHERE ID=$2 AND user_id=$3 AND archived_at IS NULL`

		res, err := tx.Exec(query, data.IsDefault, data.ID, data.UserID)
		if err != nil {
			return err
		}
		if val, err := res.RowsAffected(); val == 0 {
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

//ArchiveAddress marks the address with given addID as archived
func ArchiveAddress(addID, userID int) error {
	address, err := SelectAddressWithID(userID, addID, false)
	if err != nil {
		return err
	}
	err = database.Tx(func(tx *sqlx.Tx) error {
		query := "UPDATE address SET archived_at=now(),is_default = false WHERE ID=$1 AND user_id=$2 AND archived_at IS NULL "
		res, err := tx.Exec(query, addID, userID)
		if err != nil {
			return err
		}
		if val, err := res.RowsAffected(); val == 0 {
			if err != nil {
				return err
			}
			return sql.ErrNoRows
		}

		if address.IsDefault {
			allAddress, err := SelectAllAddress(userID)
			if err != nil {
				return err
			}
			for _, v := range allAddress {
				if v.ID != addID {
					query := `UPDATE address SET is_default = $1 WHERE id = $2`
					_, err := tx.Exec(query, true, v.ID)
					if err != nil {
						return err
					}
					break
				}
			}
		}

		return nil
	})
	return err
}

//SelectAddressWithID return details for given userID and addId
func SelectAddressWithID(userID, addID int, GetArchived bool) (models.Address, error) {
	query := `SELECT id,user_id,address_tag,address_data,is_default,lat,long,created_at,updated_at 
		FROM address 
		WHERE user_id=$1 AND id=$2 `
	if !GetArchived {
		query += "AND archived_at IS NULL"
	}

	address := models.Address{}

	err := database.YourDailyDB.Get(&address, query, userID, addID)
	return address, err
}
func SelectAddressByOrderId(orderID int) (models.Address, error) {
	query := `SELECT address.id,o.user_id,address_tag,address_data,is_default,lat,long,o.created_at,o.updated_at 
		FROM address join orders o on address.id = o.address_id 
		WHERE o.id = $1`
	address := models.Address{}

	err := database.YourDailyDB.Get(&address, query, orderID)
	return address, err
}
