package dbHelpers

import (
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/sirupsen/logrus"
)

func SelectAllActiveOrdersForStaff(staffID int, mode models.OrderMode) ([]models.StaffOrder, error) {
	query := `select u.name,
				   o.user_id,
				   o.id as order_id,
				   o.order_type,
				   o.amount,
				   u.phone,
				   o.status as status,
				   a.address_data,
				   a.lat,
				   a.long
			from orders o
					 join users u on u.id = o.user_id
					 join address a on o.address_id = a.id
			where o.staff_id = $1 and o.mode = $2::order_mode and (o.status=$3 or status = $4)
			group by order_id,u.name,u.phone, o.id, o.order_type, a.address_data, a.lat, a.long,o.created_at order by o.created_at`

	nowOrder := make([]models.StaffOrder, 0)
	err := database.YourDailyDB.Select(&nowOrder, query, staffID, mode, models.Accepted, models.OutForDelivery)
	for i, order := range nowOrder {
		nowOrder[i].Items, err = GetOrderItems(order.OrderId)
		if err != nil {
			logrus.Errorf("failed to fetch order items for order_id: %d error: %v", order.OrderId, err)
		}
	}
	return nowOrder, err
}

func UpdateLocation(staffID int, location models.GeoLocation) error {

	query := `UPDATE location set lat=$1 ,long = $2,updated_at = now() where staff_id=$3`
	res, err := database.YourDailyDB.Exec(query, location.Lat, location.Long, staffID)
	if err != nil {
		return err
	}
	if val, err := res.RowsAffected(); val == 0 {
		if err != nil {
			return err
		}
		query := `Insert Into location (staff_id,lat,long) values ($1,$2,$3)`
		_, err := database.YourDailyDB.Exec(query, staffID, location.Lat, location.Long)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAllGuest returns list of all guest users
func GetAllGuest() ([]models.UnapprovedStaff, error) {
	guests := make([]models.UnapprovedStaff, 0)
	SQL := `SELECT
			   id,
			   name,
			   phone,
			   created_at
		FROM users 
		JOIN user_permission up on users.id = up.user_id
		WHERE permission_type = $1
		ORDER BY created_at DESC`

	err := database.YourDailyDB.Select(&guests, SQL, models.Guest)

	if err != nil {
		return nil, err
	} else {
		return guests, nil
	}
}

//GetOrderByID returns and object of order for given orderId assigned to given staffID
func GetOrderByID(orderID, staffID int) (models.StaffOrder, error) {
	query := `select u.name,
       				u.phone as phone,
					o.user_id as user_id,
					o.id as order_id,
					o.order_type,
					a.address_data,
       				o.status,
					a.lat,
					a.long
				from orders o
						 join users u on u.id = o.user_id
						 join address a on o.address_id = a.id
				where  o.staff_id = $1 and o.id=$2
				group by u.name, u.phone, o.user_id, o.id, o.order_type, a.address_data, o.status, a.lat, a.long`

	orderDetails := models.StaffOrder{}
	err := database.YourDailyDB.Get(&orderDetails, query, staffID, orderID)
	if err != nil {
		return orderDetails, err
	}
	orderDetails.Items, err = GetOrderItems(orderDetails.OrderId)
	if err != nil {
		logrus.Errorf("failed to fetch order items for order_id: %d error: %v", orderDetails.OrderId, err)
	}
	return orderDetails, err
}
func GetStaffLocationByID(StaffID int) (models.LocationStatus, error) {
	query := `SELECT staff_id ,lat, long
				from location where staff_id = $1`
	staffLocation := models.LocationStatus{}
	err := database.YourDailyDB.Get(&staffLocation, query, StaffID)
	return staffLocation, err
}

func GetNewOrders(staffId int, mode models.OrderMode) ([]models.StaffOrder, error) {
	query := `select u.name,
					o.user_id as user_id,
					o.id 	  as order_id,
					o.order_type,
					a.address_data,
       				o.delivery_time,
       				o.status,
					a.lat,
					a.long
				from orders o
						 join users u on u.id = o.user_id
						 join address a on o.address_id = a.id
				where o.status = $1
				  and o.id not in (select order_id from rejected_orders where staff_id = $2)
				  AND o.staff_id is null and o.mode = $3 order by delivery_time`

	newOrders := make([]models.StaffOrder, 0)
	err := database.YourDailyDB.Select(&newOrders, query, models.Processing, staffId, mode)
	return newOrders, err
}

func GetStaffRating(staffID int) (float32, error) {
	query := `select coalesce(avg(staff_rating),0) from orders where staff_id= $1`
	var rating float32
	err := database.YourDailyDB.Get(&rating, query, staffID)
	return rating, err
}

func RejectOrder(staffID, orderID int) error {
	sql := `INSERT INTO rejected_orders (order_id, staff_id)VALUES ($1, $2)`
	_, err := database.YourDailyDB.Exec(sql, orderID, staffID)
	return err
}
