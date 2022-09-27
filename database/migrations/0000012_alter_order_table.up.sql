begin;

CREATE TYPE order_type AS ENUM (
    'now',
    'scheduled'
    );


ALTER table orders
    add column order_type order_type default 'now'::order_type;

commit;