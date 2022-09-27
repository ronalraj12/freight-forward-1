package dbHelpers

import (
	"database/sql"
	"fmt"
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/firebase"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"time"
)

//SelectOrderStatus return status and location of the order
func SelectOrderStatus(orderID int) (models.LocationStatus, error) {
	query := `SELECT o.status,l.lat,l.long ,o.staff_id
				FROM orders o LEFT JOIN location l ON o.staff_id = l.staff_id 
				WHERE o.id=$1`
	orderStatus := models.LocationStatus{}
	err := database.YourDailyDB.Get(&orderStatus, query, orderID)
	return orderStatus, err
}

//InsertIntoOrders creates a new order
func InsertIntoOrders(data models.Order) (int, error) {
	var orderID int

	err := database.Tx(func(tx *sqlx.Tx) error {
		insertOrder := `INSERT INTO orders (mode, user_id,address_id,amount, delivery_time,sm_id) 
						VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`
		err := database.YourDailyDB.Get(&orderID, insertOrder,
			data.Mode,
			data.UserID,
			data.AddressID,
			data.Amount,
			data.DeliveryTime, data.StoreMangerID)

		if err != nil {
			return err
		}

		InsertOTPQuery := `INSERT INTO order_otp (order_id,otp) VALUES($1,$2) `
		_, err = database.YourDailyDB.Exec(InsertOTPQuery, orderID, utils.GenerateOTP())
		if err != nil {
			return err
		}

		offer, err := GetActiveOffer()
		if err != nil {
			return err
		}

		for _, v := range data.Items {
			query := `INSERT INTO order_items(order_id,name,price,category,base_quantity,strikethrough_price,bucket,path,quantity, discount) (
						SELECT $1 AS order_id ,name, price, c.category, base_quantity,items.strikethrough_price, bucket, path, $2 AS quantity, $4 AS discount
												FROM items
														 JOIN categories c ON c.id = items.category
														 LEFT JOIN item_images ii ON items.id = ii.item_id
														 LEFT JOIN images i ON i.id = ii.image_id
												WHERE items.id = $3)`

			_, err = tx.Exec(query, orderID, v.Quantity, v.Id, offer.Discount)
		}
		if err != nil {
			return err
		}

		query := `
			UPDATE orders
			SET amount = (SELECT COALESCE(SUM((price - (price * discount/100)) * quantity),0)
						  FROM order_items
						  WHERE order_id = $1)
			WHERE id = $1`
		_, err = tx.Exec(query, orderID)
		if err != nil {
			return err
		}
		return nil
	})

	return orderID, err
}

//SelectLocationOfStaff returns the list of all the staff with given mode type(cart-boy/delivery-boy)
func SelectLocationOfStaff(mode models.OrderMode) ([]models.LocationStatus, error) {
	mode = mode + "-boy"
	query := `SELECT l.staff_id ,l.lat, l.long
			FROM users u JOIN user_permission up ON u.id = up.user_id RIGHT 
			JOIN LOCATION l ON up.user_id = l.staff_id
			WHERE up.permission_type = $1 AND u.enabled = TRUE`

	staffList := make([]models.LocationStatus, 0)

	err := database.YourDailyDB.Select(&staffList, query, mode)
	return staffList, err

}

//	SelectOrder return order models with all the details for given order ID
func SelectOrder(orderID, userID int) (*models.Order, error) {
	query := `SELECT id,
					   mode,
					   staff_id,
					   order_type,
					   address_id,
					   status,
					   amount,
					   delivery_time,
					   created_at,
					   updated_at,
					   otp
				FROM orders
						 LEFT JOIN order_otp ON orders.id = order_otp.order_id
				WHERE orders.id= $1
				  AND user_id = $2
				  GROUP BY orders.id, order_otp.otp`

	orderDetails := models.Order{}
	err := database.YourDailyDB.Get(&orderDetails, query, orderID, userID)
	if err != nil {
		return nil, err
	}

	orderDetails.Items, err = GetOrderItems(orderDetails.ID)
	if err != nil {
		return nil, err
	}

	return &orderDetails, err
}

func SelectAllPastOrders(userID, offset, limit int) ([]models.Order, error) {
	query := `SELECT orders.id,
					   mode,
					   staff_id,
					   order_type,
					   address_id,
					   status,
       				   u.name AS staff_name,
					   amount,
					   delivery_time,
					   orders.created_at,
					   orders.updated_at,
					   otp,
       				   staff_rating
				FROM orders 
				    LEFT JOIN users u ON orders.staff_id = u.id
						 LEFT JOIN order_otp ON orders.id = order_otp.order_id
				WHERE user_id = $1 AND orders.status IN ($4,$5,$6) 
				  GROUP BY orders.id, order_otp.otp,u.name, orders.created_at ORDER BY orders.created_at DESC OFFSET $2 LIMIT $3 `

	allOrder := make([]models.Order, 0)
	err := database.YourDailyDB.Select(&allOrder, query, userID, offset, limit, models.Declined, models.Delivered, models.Cancelled)

	for i := range allOrder {
		allOrder[i].Items, err = GetOrderItems(allOrder[i].ID)
		if err != nil {
			logrus.Errorf("failed to fetch order items for order_id: %d error: %v", allOrder[i].ID, err)
		}
	}

	for i := range allOrder {
		allOrder[i].Address, err = SelectAddressWithID(userID, allOrder[i].AddressID, true)
		if err != nil {
			logrus.Errorf("failed to fetch order address for order_id: %d error: %v", allOrder[i].ID, err)
		}
	}

	return allOrder, err
}

//SelectActiveOrders return a list of all active order for the user
func SelectActiveOrders(userID, offset, limit int) ([]models.Order, error) {
	// language=SQL
	query := `SELECT id,
				   mode,
				   user_id,
				   staff_id,
				   order_type,
				   address_id,
				   status,
				   amount,
				   delivery_time,
				   created_at,
				   updated_at,
				   otp
			FROM orders
					 LEFT JOIN order_otp ON orders.id = order_otp.order_id
			WHERE user_id = $1 AND (status =$4 OR status = $5 OR status = $6)
				AND orders.order_type = $7
			  GROUP BY orders.id, order_otp.otp, created_at 
			ORDER BY created_at 
			DESC OFFSET $2 LIMIT $3`

	allOrder := make([]models.Order, 0)
	err := database.YourDailyDB.Select(&allOrder, query, userID, offset, limit, models.OutForDelivery, models.Accepted, models.Processing, models.Now)
	if err != nil {
		return nil, err
	}
	for i := range allOrder {
		allOrder[i].Items, err = GetOrderItems(allOrder[i].ID)
		if err != nil {
			logrus.Errorf("failed to fetch allOrder[i] items for order_id: %d error: %v", allOrder[i].ID, err)
		}
		allOrder[i].Address, err = SelectAddressWithID(allOrder[i].UserID, allOrder[i].AddressID, true)
		if err != nil {
			return nil, err
		}
	}
	if len(allOrder) == 0 {
		query := `
			SELECT id,
				   mode,
				   user_id,
				   staff_id,
				   order_type,
				   address_id,
				   status,
				   amount,
				   delivery_time,
				   created_at,
				   updated_at,
				   otp
			FROM orders
					 LEFT JOIN order_otp ON orders.id = order_otp.order_id
			WHERE user_id = $1
			  AND (status = $4 OR status = $5)
			  AND (delivery_time - NOW()) <= ($7 ||' second')::INTERVAL
			  AND orders.order_type = $6
			GROUP BY orders.id, order_otp.otp, created_at
			ORDER BY created_at DESC
			OFFSET $2 LIMIT $3`

		allOrder = make([]models.Order, 0)
		err := database.YourDailyDB.Select(&allOrder, query, userID, offset, limit, models.OutForDelivery, models.Accepted, models.Scheduled, models.TimeToActivateScheduledOrder)
		if err != nil {
			return nil, err
		}
		for i := range allOrder {
			allOrder[i].Items, err = GetOrderItems(allOrder[i].ID)
			if err != nil {
				return nil, err
			}
			allOrder[i].Address, err = SelectAddressWithID(allOrder[i].UserID, allOrder[i].AddressID, true)
			if err != nil {
				return nil, err
			}
		}
	}
	return allOrder, err
}

// InsertScheduledOrder creates a new scheduled order and returns id for same
func InsertScheduledOrder(newOrder models.ScheduledOrder) (int, error) {

	var scheduleOrderID int
	txError := database.Tx(func(tx *sqlx.Tx) error {
		SQL := `INSERT INTO scheduled_orders(user_id, address_id, mode, created_at, start_date, end_date,sm_id) VALUES ($1, $2, $3, $4, $5, $6,$7) RETURNING id`
		err := tx.Get(&scheduleOrderID, SQL, newOrder.UserID, newOrder.AddressID, newOrder.Mode, time.Now(), newOrder.StartDate, newOrder.EndDate, newOrder.StoreMangerID)
		if err != nil {
			return err
		}

		for _, weekday := range newOrder.Weekdays {
			SQL = `INSERT INTO scheduled_orders_days(weekday, delivery_time, scheduled_order_id, created_at) VALUES ($1, $2, $3, $4)`
			_, err = tx.Exec(SQL, weekday, newOrder.DeliveryTime, scheduleOrderID, time.Now())
			if err != nil {
				return err
			}
		}

		for _, itemInfo := range newOrder.Items {
			SQL = `INSERT INTO scheduled_ordered_items(item_id, order_id, name, price, category,strikethrough_price, base_quantity, bucket, path, quantity) (
				   SELECT 
				          items.id,
				          $1 AS order_id,
				          name, 
				          price, 
				          c.category,
				          items.strikethrough_price,
				          base_quantity, 
				          bucket, 
				          path, 
				          $2 AS quantity
				   FROM items
				   LEFT JOIN categories c ON c.id = items.category
				   LEFT JOIN item_images ii ON items.id = ii.item_id
				   LEFT JOIN images i ON i.id = ii.image_id
				   WHERE items.id = $3
				   AND i.archived_at IS NULL
				   ORDER BY i.created_at DESC LIMIT 1)`
			_, err = tx.Exec(SQL, scheduleOrderID, itemInfo.Quantity, itemInfo.Id)
			if err != nil {
				return err
			}
		}
		//todo order amount fix in future
		SQL = `UPDATE scheduled_orders 
			   SET amount = (SELECT COALESCE(SUM(price * quantity),0) FROM scheduled_ordered_items WHERE order_id = $1) 
			   WHERE id = $1`
		result, err := tx.Exec(SQL, scheduleOrderID)
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
	})
	return scheduleOrderID, txError
}

// GetScheduledOrderById returns a scheduled order details from scheduledOrder id, user id
func GetScheduledOrderById(orderID int, userID int) (*models.ScheduledOrder, error) {

	SQL := `SELECT
				so.id,
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
			WHERE so.archived_at IS NULL
			AND so.id = $1
			AND so.user_id = $2
			GROUP BY so.id, sod.delivery_time`

	var scheduleOrder models.ScheduledOrder

	err := database.YourDailyDB.Get(&scheduleOrder, SQL, orderID, userID)
	if err != nil {
		return nil, err
	}

	SQL = `SELECT
			name,
			category,
			price,
			base_quantity,
			item_id,
       		quantity,
       		strikethrough_price
		FROM scheduled_ordered_items
		WHERE order_id = $1`

	err = database.YourDailyDB.Select(&scheduleOrder.Items, SQL, orderID)
	if err != nil {
		return nil, err
	}

	// get url for item image & store in itemInfo model todo

	return &scheduleOrder, nil
}

// GetAllScheduledOrders returns list of all scheduled orders from scheduledOrder id, user id
func GetAllScheduledOrders(userID int) ([]models.ScheduledOrder, error) {

	SQL := `SELECT
				so.id,
				so.staff_id,
				so.address_id,
				so.mode,
				so.created_at,
       			so.amount,
				ARRAY_AGG(DISTINCT sod.weekday) weekdays,
				sod.delivery_time,
       			so.start_date,
       			so.end_date
			FROM scheduled_orders so
			JOIN scheduled_orders_days sod ON so.id = sod.scheduled_order_id
			WHERE so.archived_at IS NULL
			AND so.user_id = $1
			GROUP BY so.id, sod.delivery_time, so.created_at
			ORDER BY so.created_at DESC`

	scheduledOrders := make([]models.ScheduledOrder, 0)

	err := database.YourDailyDB.Select(&scheduledOrders, SQL, userID)
	if err != nil {
		return nil, err
	}

	for i := range scheduledOrders {
		scheduledOrders[i].Items = make([]models.ItemInfo, 0)
		SQL = `SELECT
				name,
				category,
				price,
				base_quantity,
				item_id,
				quantity,
       			bucket,
       			path,
       			strikethrough_price
			 FROM scheduled_ordered_items
			 WHERE order_id = $1`

		err = database.YourDailyDB.Select(&scheduledOrders[i].Items, SQL, scheduledOrders[i].ID)
		if err != nil {
			return nil, err
		}

		for j := range scheduledOrders[i].Items {
			if scheduledOrders[i].Items[j].Bucket.Valid {
				imageInfo := models.Image{
					Bucket: scheduledOrders[i].Items[j].Bucket.String,
					Path:   scheduledOrders[i].Items[j].Path.String,
				}
				imageLink, err := firebase.GetURL(&imageInfo)
				if err != nil {
					logrus.Errorf("failed to fetch image url item id: %d  ", scheduledOrders[i].Items[j].Id)
				} else {
					scheduledOrders[i].Items[j].ItemImageLinks = append(scheduledOrders[i].Items[j].ItemImageLinks, imageLink)
				}
			}
		}
	}

	// get url for item image & store in itemInfo model todo

	for i := range scheduledOrders {
		scheduledOrders[i].Address, err = SelectAddressWithID(userID, scheduledOrders[i].AddressID, true)
		if err != nil {
			logrus.Errorf("failed to fetch order address for order_id: %d error: %v", scheduledOrders[i].ID, err)
		}
	}

	return scheduledOrders, nil
}

// ArchiveScheduledOrder archives a given scheduled order
func ArchiveScheduledOrder(orderID int, userID int) error {

	txError := database.Tx(func(tx *sqlx.Tx) error {
		var orderMode models.OrderMode
		SQL := `UPDATE scheduled_orders
			SET archived_at  = $1 
			WHERE archived_at IS NULL 
			AND id = $2 
			AND user_id = $3
			RETURNING mode`
		err := tx.Get(&orderMode, SQL, time.Now(), orderID, userID)
		if err != nil {
			return err
		} else if orderMode == "" {
			return sql.ErrNoRows
		}

		SQL = `UPDATE scheduled_orders_days
			SET archived_at  = $1 
			WHERE archived_at IS NULL 
			AND scheduled_order_id = $2`
		result, err := tx.Exec(SQL, time.Now(), orderID)
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

		if orderMode == models.DeliveryMode {
			SQL = `UPDATE scheduled_ordered_items
			SET archived_at  = $1 
			WHERE archived_at IS NULL 
			AND order_id = $2`
			result, err = tx.Exec(SQL, time.Now(), orderID)
			if err != nil {
				return err
			}
			affectedCount, err = result.RowsAffected()
			if err != nil {
				return err
			}
			if affectedCount == 0 {
				return sql.ErrNoRows
			}
		}
		return nil
	})
	return txError
}

func UpdateOrders(orderID int, staffID *int, amount *float32, userRating, staffRating *null.Float32, status *models.OrderStatus, userID *int) error {
	query := "UPDATE orders SET "
	if staffID != nil {
		query += fmt.Sprintf("staff_id = %d, ", *staffID)
	}
	if amount != nil {
		query += fmt.Sprintf("amount = %f, ", *amount)
	}
	if userRating != nil {
		if userRating.Valid {
			query += fmt.Sprintf("user_rating = %f, ", userRating.Float32)
		}
	}
	if staffRating != nil {
		if staffRating.Valid {
			query += fmt.Sprintf("staff_rating = %f, ", staffRating.Float32)
		}
	}
	if status != nil {
		query += fmt.Sprintf("status = '%s', ", *status)
	}

	query += "updated_at = now() WHERE id = $1 "
	if userID != nil {
		query += fmt.Sprintf("AND user_id = %d", *userID)
	}

	_, err := database.YourDailyDB.Exec(query, orderID)
	return err
}

//GetOTP gets the otp for the given order ID
func GetOTP(orderID int) (int, error) {
	query := `SELECT otp FROM order_otp WHERE order_id = $1`
	var OTP int
	err := database.YourDailyDB.Get(&OTP, query, orderID)
	return OTP, err
}

//DeleteOTP deletes the otp stores int the table for corresponding orderID
func DeleteOTP(orderID int) error {
	query := `DELETE FROM order_otp WHERE order_id = $1`
	_, err := database.YourDailyDB.Exec(query, orderID)
	return err
}

//AllCompletedOrders return all orders where status is delivered
func AllCompletedOrders(staffID, offset, limit int) ([]models.StaffOrder, error) {
	query := `SELECT u.name,
       			   o.amount AS amount,
				   o.status AS status,
       			   o.delivery_time,
				   o.id AS order_id,
				   o.order_type,
       			   o.user_rating,
       			   o.user_rating,
				   a.address_data
			FROM orders o
					 JOIN users u ON u.id = o.user_id
					 JOIN address a ON o.address_id = a.id
			WHERE o.staff_id = $1 AND (o.status=$2 OR o.status = $3) 
			GROUP BY order_id,u.name, o.id, o.order_type, a.address_data,o.user_rating, a.lat, a.long,o.created_at ORDER BY  o.created_at DESC OFFSET $4 LIMIT  $5`

	allOrder := make([]models.StaffOrder, 0)
	err := database.YourDailyDB.Select(&allOrder, query, staffID, models.Delivered, models.Cancelled, offset, limit)
	for i := range allOrder {
		allOrder[i].Items, err = GetOrderItems(allOrder[i].OrderId)
		if err != nil {
			logrus.Errorf("failed to fetch order items for order_id: %d error: %v", allOrder[i].OrderId, err)
		}
	}
	return allOrder, err
}

//GetUnassignedOrderCount returns count of order which are not assigned to any staff
func GetUnassignedOrderCount() (int, error) {
	var UnassignedOrder int
	query := `SELECT COUNT(*) FROM orders WHERE status = $1 AND (NOW() - created_at) > ($2 ||' second')::INTERVAL`
	err := database.YourDailyDB.Get(&UnassignedOrder, query, models.Processing, models.TimeForStaffToAcceptOrder)
	return UnassignedOrder, err
}

//GetOnGoingOrder returns the count of order which are in process of delivery
func GetOnGoingOrder(smID int) ([]models.ScheduledOrderCsv, error) {
	onGoing := make([]models.ScheduledOrderCsv, 0)
	query := `SELECT o.id           AS order_id,
				   o.mode         AS order_mode,
				   u.name         AS user_name,
				   u.phone        AS user_phone,
				   s.name         AS staff_name,
				   s.phone        AS staff_phone,
				   a.address_data AS user_address,
				   o.delivery_time,
       				o.status,
				   o.order_type,
				   o.created_at
			FROM orders o
					 JOIN address a ON a.id = o.address_id
					 JOIN users u ON a.user_id = u.id
					 LEFT JOIN users s ON o.staff_id = u.id
			WHERE (status = $1 OR status = $2) AND o.sm_id = $3`
	err := database.YourDailyDB.Select(&onGoing, query, models.Accepted, models.OutForDelivery, smID)
	if err == sql.ErrNoRows {
		return onGoing, nil
	}

	for i := range onGoing {
		var err error
		onGoing[i].Items, err = GetOrderItems(onGoing[i].OrderID)
		if err != nil {
			logrus.Errorf("failed to fetch items for order Id : %v err :%v", onGoing[i].OrderID, err)
		}
	}

	return onGoing, err
}

//GetLastWeekBookingCount Order booked in last 7 days
func GetLastWeekBookingCount() (int, error) {
	var lastWeek int
	query := `SELECT COUNT(*) FROM orders WHERE created_at>NOW()-'7 DAYS'::INTERVAL`
	err := database.YourDailyDB.Get(&lastWeek, query)
	return lastWeek, err
}

//GetNSGStats return a array of number of now and scheduled order for lat 14 days
func GetNSGStats(days int) ([]models.OrderNSStats, error) {

	dayStr := fmt.Sprintf("%d DAYS", days-1)
	orderStats := make([]models.OrderNSStats, 0)
	query := `SELECT day_offset.date::DATE AS order_date,
       scheduled.count       AS scheduled_orders,
       now_orders.count      AS now_orders
FROM (
         SELECT TO_CHAR(DATE_TRUNC('day', (offs)), 'YYYY-MM-DD') AS date
         FROM GENERATE_SERIES(
                      (NOW() - $1::INTERVAL)::DATE,
                      NOW()::DATE,
                      '1 day'::INTERVAL) AS offs
     ) AS day_offset
         LEFT JOIN LATERAL (
    SELECT COUNT(id) AS count
    FROM orders
    WHERE orders.order_type = 'scheduled' 
      AND orders.created_at::DATE = day_offset.date::DATE
    ) scheduled ON TRUE
         LEFT JOIN LATERAL (
    SELECT COUNT(id) AS count
    FROM orders
    WHERE orders.order_type = 'now'
      AND orders.created_at::DATE = day_offset.date::DATE
    ) now_orders ON TRUE
ORDER BY order_date`

	err := database.YourDailyDB.Select(&orderStats, query, dayStr)

	return orderStats, err

}

func GetOrderAcceptedStats(days int) ([]models.OrderADStats, error) {
	dayStr := fmt.Sprintf("%d DAYS", days-1)
	orderADStats := make([]models.OrderADStats, 0)
	query := `SELECT day_offset.date::DATE AS order_date,
       scheduled.count       AS accepted_orders,
       now_orders.count      AS declined_orders
FROM (
         SELECT TO_CHAR(DATE_TRUNC('day', (offs)), 'YYYY-MM-DD') AS date
         FROM GENERATE_SERIES(
                      (NOW() - $1::INTERVAL)::DATE,
                      NOW()::DATE,
                      '1 day'::INTERVAL) AS offs
     ) AS day_offset
         LEFT JOIN LATERAL (
    SELECT COUNT(id) AS count
    FROM orders
    WHERE orders.status !='declined' AND orders.status!='processing'
      AND orders.created_at::DATE = day_offset.date::DATE
    ) scheduled ON TRUE
         LEFT JOIN LATERAL (
    SELECT COUNT(id) AS count
    FROM orders
    WHERE orders.status = 'declined'
      AND orders.created_at::DATE = day_offset.date::DATE
    ) now_orders ON TRUE
ORDER BY order_date`
	err := database.YourDailyDB.Select(&orderADStats, query, dayStr)

	return orderADStats, err
}

func GetOrdersForStatus(status models.OrderStatus) ([]models.DeniedUnassignedOrders, error) {
	query := `SELECT o.id AS order_id,
				   o.order_type,
				   o.mode,
				   o.delivery_time,
				   u.phone,
				   a.address_data,
				   o.status
			FROM orders o
					 JOIN users u ON u.id = o.user_id
					 JOIN address a ON o.address_id = a.id
			WHERE o.status = $1
			  AND (NOW() - o.created_at) > ($2 ||' second')::INTERVAL
			GROUP BY order_id, u.name, u.phone, o.order_type, o.id, a.address_data, o.status, o.created_at
			ORDER BY o.created_at DESC`

	orderDetails := make([]models.DeniedUnassignedOrders, 0)
	err := database.YourDailyDB.Select(&orderDetails, query, status, models.TimeForStaffToAcceptOrder)
	if err != nil {
		return orderDetails, err
	}
	//for _, v := range orderDetails {
	//	SQL := `SELECT
	//			name,
	//			category,
	//			price,
	//			base_quantity,
	//			quantity
	//		 FROM order_items
	//		 WHERE order_id = $1`
	//
	//	err = database.YourDailyDB.Select(&v.Items, SQL, v.OrderId)
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	return orderDetails, err
}

func GetOrderCountForStatus(status models.OrderStatus) (int, error) {
	query := `SELECT COUNT(*)
				FROM orders o
				WHERE o.status = $1 AND (NOW() - o.created_at) > ($2 ||' second')::INTERVAL`

	var orderCount int
	err := database.YourDailyDB.Get(&orderCount, query, status, models.TimeForStaffToAcceptOrder)
	return orderCount, err
}

func MoveScheduledOrders() error {

	// get (id, mode) of all orders scheduled for today
	SQL := `SELECT
            so.id,
            so.mode,
       		so.user_id AS user_id
          FROM scheduled_orders so
          JOIN scheduled_orders_days sod ON so.id = sod.scheduled_order_id
          WHERE sod.weekday = $1
          AND so.start_date::DATE <= NOW()::DATE
          AND (so.end_date::DATE IS NULL OR so.end_date::DATE >= NOW()::DATE)
          AND sod.archived_at IS NULL AND so.archived_at IS NULL`

	var eligibleScheduledOrders []models.OrderResponse
	err := database.YourDailyDB.Select(&eligibleScheduledOrders, SQL, time.Now().Weekday().String())
	if err != nil {
		return err
	}

	// run transaction for each order & perform required operations
	for i := range eligibleScheduledOrders {

		txError := database.Tx(func(tx *sqlx.Tx) error {
			flagCount, _, err := GetFlagCountAndLastOrderStatus(eligibleScheduledOrders[i].UserID)
			if err != nil {
				logrus.Errorf("MoveScheduledOrder:%v", err)
				return err
			}
			if flagCount >= models.MaxFlagCount {
				logrus.Errorf("moveScheduledOrder : User account blocked %d :", eligibleScheduledOrders[i].UserID)
			}
			// move order details
			SQL := `INSERT INTO orders(mode, user_id, staff_id, address_id, status, delivery_time, order_type, amount,sm_id)
								   (SELECT
									   so.mode,
									   so.user_id,
									   so.staff_id,
									   so.address_id,
									   $1 AS status,
									   TO_TIMESTAMP(EXTRACT(EPOCH FROM NOW()::DATE) + EXTRACT(EPOCH FROM (sod.delivery_time - sod.delivery_time::DATE))) AS delivery_time,
									   $2 AS order_type,
									   $3 AS amount ,
									   so.sm_id
									FROM scheduled_orders so
									JOIN scheduled_orders_days sod ON so.id = sod.scheduled_order_id	
									WHERE so.id = $4
									GROUP BY so.id, sod.delivery_time)
									RETURNING id, mode`

			var newlyMovedOrder models.OrderResponse
			err = tx.Get(&newlyMovedOrder, SQL, models.ScheduledOrderStatus, models.Scheduled, 0.0, eligibleScheduledOrders[i].OrderID)
			if err != nil {
				return err
			}

			// generate OTP for these newly moved order
			InsertOTPQuery := `INSERT INTO order_otp(order_id, otp) VALUES($1, $2)`
			_, err = tx.Exec(InsertOTPQuery, newlyMovedOrder.OrderID, utils.GenerateOTP())
			if err != nil {
				return err
			}

			if newlyMovedOrder.Mode == models.DeliveryMode {

				// get current discount
				offer, err := GetActiveOffer()
				if err != nil {
					return err
				}

				// move item details
				SQL := `INSERT INTO order_items(name, order_id, price, category, base_quantity,strikethrough_price, quantity, bucket, path, discount)
                 SELECT
                     soi.name,
                     $1 AS order_id,
                     i.price,
                     soi.category,
                     soi.base_quantity,
                     i.strikethrough_price,
                     quantity,
                     bucket,
                     path,
                     $2 AS discount
                 FROM scheduled_ordered_items soi
                 JOIN items i ON soi.item_id = i.id
                 AND soi.order_id = $3`

				_, err = tx.Exec(SQL, newlyMovedOrder.OrderID, offer.Discount, eligibleScheduledOrders[i].OrderID)
				if err != nil {
					return err
				}

				// calculate amount
				SQL = `UPDATE orders
                  SET amount = (SELECT (SUM((price - (price * discount/100)) * quantity)) FROM order_items WHERE order_id = $1) 
                  WHERE id = $1`
				result, err := tx.Exec(SQL, newlyMovedOrder.OrderID)
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
			}
			return nil
		})
		if txError != nil {
			logrus.Errorf("failed to move an scheduled order having id: %d with error: %s, skipped!", eligibleScheduledOrders[i].OrderID, txError)
		}
	}
	return nil
}

func CheckOrderStatus(staffID, orderId int) (int, error) {
	query := `SELECT COUNT(id)
				FROM orders
				WHERE staff_id = $1
				  AND status = $2 AND id <> $3`
	var count int
	err := database.YourDailyDB.Get(&count, query, staffID, models.OutForDelivery, orderId)
	return count, err
}

// GetCurrentOrderStatus returns status for a given order id
func GetCurrentOrderStatus(orderID int) (models.OrderStatus, error) {
	SQL := `SELECT status
				FROM orders
				WHERE id = $1`

	var status models.OrderStatus
	err := database.YourDailyDB.Get(&status, SQL, orderID)
	return status, err
}

func GetScheduledOrderToSendNotification() {
	//	query := `
	//
	//`
}
