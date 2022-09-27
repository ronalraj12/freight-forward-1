package dbHelpers

import (
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/models"
)

func InsertDisputedOrder(orderID, userId int) error {
	query := `Insert into disputed_orders(order_id, disputed_at, disputed_by)
			values ($1,now(),$2)`

	_, err := database.YourDailyDB.Exec(query, orderID, userId)
	return err
}

func GetAllDisputedOrders() ([]models.DisputedOrder, error) {
	query := `select dis.order_id   as order_id,
					   u.phone        as user_phone,
					   a.address_data as address_Data,
					   disputed_at,
					   resolved_at
				from disputed_orders dis
						 join users as u on dis.disputed_by = u.id
						 join orders o on dis.order_id = o.id
						 join address a on o.address_id = a.id`
	disOrder := make([]models.DisputedOrder, 0)
	err := database.YourDailyDB.Select(&disOrder, query)
	return disOrder, err
}

func GetDisputedOrderInfo(OrderId int) (models.DisputedOrderInfo, error) {
	query := `select dis.order_id,
				   o.mode,
				   u.name         as user_name,
				   u.phone        as user_phone,
				   st.name        as staff_name,
				   st.phone       as staff_phone,
       			   o.amount
			from disputed_orders dis
					 join users as u on dis.disputed_by = u.id
					 join orders o on dis.order_id = o.id
					 join address a on o.address_id = a.id
					 join users st on st.id = o.staff_id where order_id= $1`
	disOrder := models.DisputedOrderInfo{}
	err := database.YourDailyDB.Get(&disOrder, query, OrderId)
	if err != nil {
		return disOrder, err
	}
	SQL := `SELECT
				name,
				category,
				price,
				base_quantity,
				quantity
			  FROM order_items
			  WHERE order_id = $1`
	items := make([]models.ItemInfo, 0)
	err = database.YourDailyDB.Select(&items, SQL, OrderId)
	disOrder.Items = items
	if err != nil {
		return disOrder, err
	}
	return disOrder, nil
}
