BEGIN;

ALTER TABLE orders
    ADD COLUMN sm_id int;

ALTER TABLE scheduled_orders
    ADD COLUMN sm_id int;

COMMIT;