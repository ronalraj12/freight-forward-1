ALTER TABLE scheduled_orders
    ADD COLUMN start_date timestamp,
    ADD COLUMN end_date timestamp
    check(start_date <= end_date);