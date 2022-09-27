BEGIN;

CREATE TYPE order_mode AS ENUM (
    'cart',
    'delivery'
    );

CREATE TYPE order_status AS ENUM (
    'processing',
    'accepted',
    'outForDelivery',
    'delivered',
    'cancelled'
    );

CREATE TABLE orders
(
    id            SERIAL PRIMARY KEY,
    mode          order_mode,
    user_id       INT REFERENCES users (id),
    staff_id      INT REFERENCES users (id),
    address_id    INT REFERENCES address (id),
    status        order_status DEFAULT 'processing'::order_status,
    order_otp     INT NOT NULL,
    amount        DECIMAL,
    user_rating   INT          DEFAULT 0 CHECK (user_rating >= 0 AND user_rating <= 5),
    staff_rating  INT          DEFAULT 0 CHECK (staff_rating >= 0 AND staff_rating <= 5),
    delivery_time TIMESTAMP,
    items         TEXT,
    created_at    TIMESTAMP    DEFAULT now(),
    archived_at   TIMESTAMP,
    updated_at    TIMESTAMP
);

commit;