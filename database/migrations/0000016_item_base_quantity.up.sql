BEGIN;

ALTER TABLE items ADD COLUMN base_quantity TEXT;

ALTER TABLE scheduled_orders ADD COLUMN amount DECIMAL;

COMMIT;
