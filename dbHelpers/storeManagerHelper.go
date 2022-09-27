package dbHelpers

import (
	"database/sql"
	"fmt"
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"time"
)

// UpdateStaffPermission adds new permission (user+cartboy OR user+deliveryboy) for a given guest & removes guest permission
func UpdateStaffPermission(userID int, permission models.UserPermission) error {
	txError := database.Tx(func(tx *sqlx.Tx) error {
		// remove guest permission
		SQL := `DELETE FROM user_permission WHERE user_id = $1 AND permission_type = $2`
		result, err := tx.Exec(SQL, userID, models.Guest)
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

		// add required permission(user+cart/delivery)
		SQL = `INSERT INTO user_permission(user_id, permission_type) VALUES ($1, $2), ($3, $4)`
		_, err = tx.Exec(SQL, userID, permission, userID, models.DefaultUser)
		if err != nil {
			return err
		}
		return nil
	})
	return txError
}

// ArchiveStaffByUserID archives a staff member(cart/delivery boy)
func ArchiveStaffByUserID(userID int) error {
	SQL := `UPDATE users
			SET archived_at = $1
			WHERE archived_at IS NULL
			AND id = $2
			AND EXISTS (SELECT 1 FROM user_permission WHERE user_id = $3 AND (permission_type = $4 OR permission_type = $5));`
	result, err := database.YourDailyDB.Exec(SQL, time.Now(), userID, userID, models.CartBoy, models.DeliveryBoy)
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

	// todo its possible that this staff member has some orders assigned, need to handle that later

	return nil
}

func GetStaffStats(staffType string, staffID *int) ([]models.DashBoardStaffDetails, error) {

	query := `SELECT id,
       				 sd.name,
				     sd.phone,
      				 sd.enabled,
				     sd.created_at,
				     sd.total_order,
				     sd.total_amount,
				     sd.avg_rating,
				     denied_orders,
				     cancelled_orders
				FROM (
						 SELECT u.id AS id,
								name,
						        u.enabled,
								phone,
								u.created_at        AS created_at,
								COALESCE(COUNT(o.id),0)         AS total_order,
								COALESCE(SUM(amount),0)         AS total_amount,
								COALESCE(AVG(o.staff_rating),0) AS avg_rating
						 FROM users u
								  LEFT JOIN user_permission up ON u.id = up.user_id
								  LEFT JOIN orders o ON u.id = o.staff_id
						 WHERE up.permission_type = $1
						 GROUP BY u.id, u.name, u.phone, u.created_at
					 ) AS sd
						 LEFT JOIN LATERAL (
					SELECT COUNT(id) AS denied_orders
					FROM orders
					WHERE orders.status = $2
					  AND sd.id = orders.staff_id
					) declined ON TRUE
						 LEFT JOIN LATERAL (
					SELECT COUNT(id) AS cancelled_orders
					FROM orders
					WHERE orders.status = $3
					  AND sd.id = orders.staff_id
					) cancelled ON TRUE `
	if staffID != nil {
		query += fmt.Sprintf(" where id = %d", *staffID)
	}
	query += " ORDER BY id"
	staffStats := make([]models.DashBoardStaffDetails, 0)
	err := database.YourDailyDB.Select(&staffStats, query, staffType, models.Declined, models.Cancelled)
	return staffStats, err
}

func GetAllUserStats(userID *int) ([]models.DashBoardUserDetails, error) {
	query := `
		SELECT sd.id    AS id,
			   sd.flags AS flag_count,
			   sd.name,
			   sd.phone,
			   sd.created_at,
			   sd.total_amount,
			   default_address,
			   default_lat,
			   default_long,
			   sd.total_order,
			   sd.avg_rating,
			   denied_orders,
			   canceled_orders
		
		FROM (
				 SELECT u.id,
						name,
						phone,
						flags,
						u.created_at,
						COUNT(o.id)                            AS total_order,
						COALESCE(SUM(amount), 0)               AS total_amount,
						COALESCE(AVG(o.user_rating), 0)        AS avg_rating,
						ARRAY_AGG(DISTINCT up.permission_type) AS permissions
				 FROM users u
						  LEFT JOIN user_permission up ON u.id = up.user_id
						  LEFT JOIN orders o ON u.id = o.user_id
				 WHERE o.archived_at IS NULL
				 GROUP BY u.id, name, phone
			 ) AS sd
				 LEFT JOIN LATERAL (
			SELECT address_data AS default_Address, lat AS default_lat, long AS default_long
			FROM address
			WHERE address.is_default = TRUE
			  AND sd.id = address.user_id
			) address_table ON TRUE
				 LEFT JOIN LATERAL (
			SELECT COUNT(id) AS denied_orders
			FROM orders
			WHERE orders.status = $1
			  AND orders.user_id = id
			) declined_table ON TRUE
				 LEFT JOIN LATERAL (
			SELECT COUNT(id) AS canceled_orders
			FROM orders
			WHERE orders.status = $2
			  AND sd.id = orders.user_id
			) cancelled_table ON TRUE
		WHERE ARRAY_LENGTH(permissions, 1) = 1
`
	if userID != nil {
		query += fmt.Sprintf(" and id = %d", *userID)
	} else {
		query += ` and id in (SELECT id
							FROM users
							WHERE id NOT IN (
								SELECT u.id
								FROM users u
										 JOIN user_permission up ON u.id = up.user_id
								WHERE up.permission_type <> 'user'
							) and archived_at is null
)`
	}
	query += " ORDER BY id"

	userStats := make([]models.DashBoardUserDetails, 0)
	err := database.YourDailyDB.Select(&userStats, query, models.Declined, models.Cancelled)
	if err != nil {
		return userStats, err
	}

	for i := range userStats {
		SQL := `SELECT ARRAY(SELECT oi.name
		FROM users u
			JOIN orders o ON u.id = o.user_id
			JOIN order_items oi ON o.id = oi.order_id
		WHERE u.id = $1
		GROUP BY oi.name ORDER BY (COALESCE( COUNT(u.id),0)) DESC LIMIT 3) AS top_three_items`

		err = database.YourDailyDB.Get(&userStats[i].TopThreeItems, SQL, userStats[i].ID)
		if err != nil {
			logrus.Errorf("failed to fetch user stats for userID := %d", userStats[i].ID)
		}

		SQL = `
				SELECT address_id AS id, a.address_data, a.lat, a.long
				FROM users u
						 JOIN orders o ON u.id = o.user_id
						 JOIN address a ON a.id = o.address_id
				WHERE u.id = $1
				GROUP BY o.address_id, a.address_data, address_id, a.long, a.lat
				ORDER BY COUNT(o.address_id) DESC
				LIMIT 3`
		locations := make([]models.Address, 0)
		err = database.YourDailyDB.Select(&locations, SQL, userStats[i].ID)
		if err != nil {
			logrus.Errorf("failed to fetch user stats for userID := %d", userStats[i].ID)
		}
		userStats[i].TopThreeLocation = locations
	}

	return userStats, nil
}

func GetOrderInfo(orderID int) (*models.OrderInfo, error) {
	SQL := `SELECT
       mode,
       user_id,
       address_id
   FROM orders
   WHERE id = $1
   AND status = $2`

	var order models.OrderInfo
	err := database.YourDailyDB.Get(&order, SQL, orderID, models.Processing)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func MarkAsResolved(orderId, resolverId int) error {
	query := `UPDATE disputed_orders
			SET resolved_at=NOW(),
				resolved_by=$1 WHERE  order_id = $2`
	_, err := database.YourDailyDB.Exec(query, resolverId, orderId)
	return err
}

func RemoveGuest(userID int) error {
	query := `DELETE 
		FROM user_permission 
		WHERE user_id = $1 
		AND permission_type = $2`
	result, err := database.YourDailyDB.Exec(query, userID, models.Guest)
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

func GetStoreManagerByEmail(email string) (*models.User, error) {
	query := `SELECT id, name, email,password, phone,permission_type
FROM users JOIN user_permission up ON users.id = up.user_id
WHERE email =  $1`
	var user models.User
	err := database.YourDailyDB.Get(&user, query, email)
	if err != nil {
		return nil, err
	}

	return &user, err
}

// StoreImageInfo stores new image of given type, returns image id
func StoreImageInfo(bucket string, path string, imageType string) (int, error) {
	SQL := `INSERT INTO images(type, bucket, path, created_at) VALUES ($1, $2, $3, $4) RETURNING id`
	var imageID int
	err := database.YourDailyDB.Get(&imageID, SQL, imageType, bucket, path, time.Now())
	if err != nil {
		return -1, err
	}
	return imageID, nil
}

func CreateNewOffer(title, description string, discount int, imageID null.Int) error {
	txError := database.Tx(func(tx *sqlx.Tx) error {
		query := `UPDATE offers 
		SET archived_at = $1
		WHERE archived_at IS NULL`
		_, err := tx.Exec(query, time.Now())
		if err != nil {
			return err
		}

		query = `INSERT INTO offers(title, description, discount, image_id) VALUES($1, $2, $3, $4)`
		_, err = tx.Exec(query, title, description, discount, imageID)
		return err
	})
	return txError
}

func ArchiveActiveOffer() error {
	query := `UPDATE offers 
		SET archived_at = $1
		WHERE archived_at IS NULL`

	result, err := database.YourDailyDB.Exec(query, time.Now())
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

func GetActiveOffer() (*models.Offer, error) {
	query := `SELECT 
				id,
				title,
				description,
       			discount,
				image_id
			FROM offers 
			WHERE archived_at IS NULL`

	activeOffer := models.Offer{}
	err := database.YourDailyDB.Get(&activeOffer, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return &activeOffer, nil
		}
		return nil, err
	}
	return &activeOffer, nil
}

// GetImageInfoByOfferID returns image info - bucket & path
func GetImageInfoByOfferID(offerID int) (*models.Image, error) {
	SQL := `SELECT i.bucket, i.path
			FROM images i
			JOIN offers o
			ON i.id = o.image_id
			WHERE o.id = $1;`

	var imageInfo models.Image
	err := database.YourDailyDB.Get(&imageInfo, SQL, offerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &imageInfo, nil
}

func AdminContactInfo() (string, error) {
	query := `SELECT phone
				FROM users
						 JOIN user_permission up ON users.id = up.user_id
				WHERE permission_type = 'store-manager'`
	var phone string
	err := database.YourDailyDB.Get(&phone, query)
	return phone, err
}

func GetNewOrdersForSm() ([]models.StaffOrder, error) {
	//language=SQL
	query := `
		SELECT orders.id AS order_id, address_data, u.phone, order_type
		FROM orders
				 JOIN users u ON u.id = orders.user_id
				 JOIN address a ON a.id = orders.address_id
		WHERE status = $1
		  AND ((order_type = $2 AND (NOW() - orders.delivery_time) > ($4 || 'second')::INTERVAL)
			OR (order_type = $3 AND (orders.delivery_time - NOW() < ($5 || 'second')::INTERVAL)))
`
	newOrders := make([]models.StaffOrder, 0)
	err := database.YourDailyDB.Select(&newOrders, query, models.Processing, models.Now, models.Scheduled, models.TimeForStaffToAcceptOrder, models.TimeToActivateScheduledOrder-30)
	if err == sql.ErrNoRows {
		return newOrders, nil
	}
	return newOrders, err
}

func StoreManagerNearMe(lat, long float64) (int, error) {
	query := `
		SELECT u.id, l.lat, l.long
		FROM users u
		JOIN user_permission up ON u.id = up.user_id
		JOIN location l ON u.id = l.staff_id
		WHERE up.permission_type = 'store-manager'
`
	sms := make([]models.Location, 0)
	err := database.YourDailyDB.Select(&sms, query)
	if err != nil {

		return 0, err
	}
	for _, sm := range sms {
		if sm.Long.Valid && sm.Lat.Valid {
			dist := utils.GeoDistance(long, lat, sm.Long.Float64, sm.Lat.Float64)
			if dist <= float64(models.RadiusForSearch) {
				return sm.Id, nil
			}
		}

	}
	err = fmt.Errorf("Launching soon in your area!!!")
	return 0, err
}

func GetScheduledOrderInRange(startDate time.Time, endDate time.Time) ([]models.ScheduledOrderCsv, error) {

	query := `
	SELECT so.id                                                                           AS order_id,
		   so.mode                                                                         AS order_mode,
		   u.name                                                                          AS user_name,
		   u.phone                                                                         AS user_phone,
		   s.name                                                                          AS staff_name,
		   s.phone                                                                         AS staff_phone,
		   a.address_data                                                                  AS user_address,
		   TO_TIMESTAMP(EXTRACT(EPOCH FROM days.date::DATE) +
						EXTRACT(EPOCH FROM (sod.delivery_time - sod.delivery_time::DATE))) AS delivery_time,
		   $1                                                                              AS order_type,
		   so.start_date,
		   so.end_date,
	       so.created_at
	FROM scheduled_orders so
			 JOIN scheduled_orders_days sod
				  ON so.id = sod.scheduled_order_id
			 JOIN (SELECT date
				   FROM GENERATE_SERIES(
								($2)::DATE,
								($3)::DATE,
								'1 day'::INTERVAL) AS date) AS days
				  ON TO_CHAR(date, 'Day')::TEXT LIKE '%' || sod.weekday::TEXT || '%'
			 JOIN address a ON a.id = so.address_id
			 LEFT JOIN users s ON s.id = so.staff_id
			 JOIN users u ON u.id = a.user_id
	WHERE date <= end_date
	  AND date >= start_date
	  AND so.archived_at IS NULL
	ORDER BY delivery_time
`
	orders := make([]models.ScheduledOrderCsv, 0)
	err := database.YourDailyDB.Select(&orders, query, models.Scheduled, startDate, endDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return orders, nil
		}
		return orders, err
	}

	for i := range orders {
		var err error
		orders[i].Items, err = GetScheduledOrderItems(orders[i].OrderID)
		if err != nil {
			logrus.Errorf("failed to fetch items for order Id : %v err :%v", orders[i].OrderID, err)
		}
	}

	return orders, nil
}

func GetOrderInDateRange(startDate time.Time, endDate time.Time, smID int) ([]models.ScheduledOrderCsv, error) {
	query := `
			SELECT o.id           AS order_id,
				   o.mode         AS order_mode,
			       o.status AS status,
				   u.name         AS user_name,
				   u.phone        AS user_phone,
				   s.name         AS staff_name,
				   s.phone        AS staff_phone,
				   a.address_data AS user_address,
				   o.delivery_time,
				   o.order_type,
				   o.created_at
			FROM orders o
					 JOIN address a ON a.id = o.address_id
					 JOIN users u ON a.user_id = u.id
					 LEFT JOIN users s ON o.staff_id = s.id
			WHERE  o.sm_id = $1 AND  o.created_at::DATE >= $2::DATE AND o.created_at::DATE <=$3::DATE 
			ORDER BY  o.created_at
`
	orders := make([]models.ScheduledOrderCsv, 0)
	err := database.YourDailyDB.Select(&orders, query, smID, startDate, endDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return orders, nil
		}
		return orders, err
	}

	for i := range orders {
		var err error
		orders[i].Items, err = GetOrderItems(orders[i].OrderID)
		if err != nil {
			logrus.Errorf("failed to fetch items for order Id : %v err :%v", orders[i].OrderID, err)
		}
	}

	return orders, nil

}
func GetAllScheduledOrdersForSm(smId int) ([]models.ScheduledOrder, error) {
	SQL := `SELECT so.id,
				   so.user_id,
				   u.phone AS             user_phone,
				   u.name  AS             user_name,
				   s.phone AS             staff_name,
				   s.name  AS             staff_name,
				   so.staff_id,
				   so.address_id,
				   so.mode,
				   so.created_at,
				   so.amount,
				   ARRAY_AGG(sod.weekday) weekdays,
				   sod.delivery_time,
				   so.start_date,
				   so.end_date
			FROM scheduled_orders so
					 JOIN scheduled_orders_days sod ON so.id = sod.scheduled_order_id
					 JOIN address a ON a.id = so.address_id
					 JOIN users u ON so.user_id = u.id
					 LEFT JOIN users s ON so.staff_id = s.id
			WHERE so.archived_at IS NULL
			  AND sm_id = $1
			  AND(end_date is null or end_date::date > now() + interval '1 day'::interval)
			GROUP BY so.id, so.created_at, a.address_data, so.staff_id, so.address_id, so.id, so.mode, so.created_at, so.amount,
					 sod.delivery_time, so.start_date, so.end_date, so.user_id, u.phone, u.name, s.phone, s.name, so.staff_id,
					 so.address_id, so.mode, so.created_at, so.amount, sod.delivery_time, so.start_date, so.id
			ORDER BY so.id ASC`

	scheduledOrders := make([]models.ScheduledOrder, 0)

	err := database.YourDailyDB.Select(&scheduledOrders, SQL, smId)
	if err != nil {
		if err == sql.ErrNoRows {
			return scheduledOrders, nil
		}
		return nil, err
	}
	for i := range scheduledOrders {
		scheduledOrders[i].Address, err = SelectAddressWithID(scheduledOrders[i].UserID, scheduledOrders[i].AddressID, true)
		if err != nil && err != sql.ErrNoRows {
			logrus.Errorf("unable to get scheduled address for orderID = %d ", scheduledOrders[i].ID)
		}
		scheduledOrders[i].Items, err = GetScheduledOrderItems(scheduledOrders[i].ID)
		if err != nil && err != sql.ErrNoRows {
			logrus.Errorf("unable to get scheduled order items for orderID = %d ", scheduledOrders[i].ID)
		}
	}
	return scheduledOrders, nil
}

func ArchiveScheduledOrderWithOrderID(orderID, smID int) (int, error) {
	var userID int
	err := database.Tx(func(tx *sqlx.Tx) error {
		query := `
		UPDATE scheduled_orders_days
				SET archived_at = NOW()
				WHERE id = $1`
		err := database.YourDailyDB.Get(&userID, query, orderID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		query = `
		UPDATE scheduled_orders
				SET archived_at = NOW()
				WHERE id = $1
				  AND sm_id = $2`
		err = database.YourDailyDB.Get(&userID, query, orderID, smID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		return nil
	})
	return userID, err

}

func EnableDisableStaff(status bool, userID int) error {

	query := `
			UPDATE users SET enabled = $1 WHERE id = $2 
`
	_, err := database.YourDailyDB.Exec(query, status, userID)
	return err
}

func ChangeStaffRole(staffID int, newRole string) error {
	err := database.Tx(func(tx *sqlx.Tx) error {
		query := `DELETE 
					FROM user_permission 
					WHERE user_id=$1 AND  (permission_type = 'cart-boy' OR permission_type ='delivery-boy')`
		_, err := tx.Exec(query, staffID)
		if err != nil {
			return err
		}
		query = `INSERT INTO user_permission (user_id,permission_type) VALUES ($1,$2)`
		_, err = tx.Exec(query, staffID, newRole)
		return err
	})
	return err

}
