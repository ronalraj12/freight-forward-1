BEGIN;

ALTER TABLE scheduled_orders_days
    DROP COLUMN delivery_time;

ALTER TABLE scheduled_orders_days
    ADD COLUMN delivery_time timestamptz;

END;