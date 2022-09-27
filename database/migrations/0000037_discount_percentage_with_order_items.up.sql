BEGIN;

ALTER TABLE order_items
    ADD COLUMN discount int CHECK ( discount >= 0 AND discount <= 100 ) DEFAULT 0;

ALTER TABLE orders
    DROP COLUMN user_rating,
    DROP COLUMN staff_rating;

ALTER TABLE orders
    ADD COLUMN user_rating  decimal DEFAULT NULL CHECK ( user_rating IS NULL OR (user_rating >= 0 AND user_rating <= 5) ),
    ADD COLUMN staff_rating decimal DEFAULT NULL CHECK ( staff_rating IS NULL OR (staff_rating >= 0 AND staff_rating <= 5) );

COMMIT;