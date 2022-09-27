BEGIN;

CREATE TABLE weekdays
(
    weekday_num INT PRIMARY KEY,
    day_of_week CHAR(3) NOT NULL,
    UNIQUE (day_of_week)
);

INSERT INTO weekdays
VALUES (1, 'sun');
INSERT INTO weekdays
VALUES (2, 'mon');
INSERT INTO weekdays
VALUES (3, 'tue');
INSERT INTO weekdays
VALUES (4, 'wed');
INSERT INTO weekdays
VALUES (5, 'thu');
INSERT INTO weekdays
VALUES (6, 'fri');
INSERT INTO weekdays
VALUES (7, 'sat');

CREATE TABLE scheduled_orders
(
    id          SERIAL PRIMARY KEY,
    user_id     INT REFERENCES users (id),
    staff_id    INT REFERENCES users (id),
    address_id  INT REFERENCES address (id),
    mode        order_mode,
    items       TEXT,
    created_at  TIMESTAMP DEFAULT now(),
    archived_at TIMESTAMP,
    updated_at  TIMESTAMP,
    UNIQUE (user_id, mode)

);

CREATE TABLE scheduled_orders_days
(
    id                 SERIAL PRIMARY KEY,
    day_of_week        CHAR(3) REFERENCES weekdays (day_of_week),
    delivery_time      TIMESTAMP,
    scheduled_order_id INT REFERENCES scheduled_orders (id),
    created_at         TIMESTAMP DEFAULT now(),
    archived_at        TIMESTAMP,
    updated_at         TIMESTAMP
);

COMMIT;
