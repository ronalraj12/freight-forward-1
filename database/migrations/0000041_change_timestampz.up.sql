BEGIN;

ALTER TABLE scheduled_orders
    alter column start_date type timestamptz;

ALTER TABLE scheduled_orders
    alter column end_date type timestamptz;

COMMIT;