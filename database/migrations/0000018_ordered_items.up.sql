BEGIN;

ALTER TABLE scheduled_orders DROP COLUMN items;

CREATE TABLE scheduled_ordered_items(
    id SERIAL PRIMARY KEY,
    order_id INT REFERENCES scheduled_orders(id),
    item_id INT REFERENCES items(id),
    quantity INT CHECK (quantity > 0),
    current_price DECIMAL,
    created_at TIMESTAMP DEFAULT now(),
    archived_at TIMESTAMP,
    updated_at TIMESTAMP
);

ALTER TABLE scheduled_orders
    DROP CONSTRAINT scheduled_orders_user_id_mode_key;

COMMIT;
