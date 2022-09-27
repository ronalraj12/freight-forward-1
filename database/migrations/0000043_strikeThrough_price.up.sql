BEGIN;

ALTER TABLE items
    ADD COLUMN strikethrough_price decimal(20, 3);

ALTER TABLE order_items
    ADD COLUMN strikethrough_price decimal(20, 3);

ALTER TABLE scheduled_ordered_items
    ADD COLUMN strikethrough_price decimal(20, 3);

COMMIT;