CREATE TABLE rejected_orders
(
    id       serial PRIMARY KEY,
    order_id int NOT NULL,
    staff_id int NOT NULL
);