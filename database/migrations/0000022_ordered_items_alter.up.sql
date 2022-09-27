BEGIN;

ALTER TABLE scheduled_ordered_items
ADD COLUMN
    name        text,
ADD COLUMN
    category    text,
ADD COLUMN
    bucket      text,
ADD COLUMN
    path        text;

ALTER TABLE scheduled_ordered_items
    RENAME COLUMN current_price TO price;

COMMIT;