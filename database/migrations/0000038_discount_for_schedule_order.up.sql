ALTER TABLE scheduled_ordered_items
    ADD COLUMN discount int CHECK ( discount >= 0 AND discount <= 100 ) DEFAULT 0;
