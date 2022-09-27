begin;

ALTER TABLE scheduled_orders_days DROP COLUMN day_of_week;
DROP TABLE weekdays;

CREATE TYPE weekdays AS ENUM (
    'Monday',
    'Tuesday',
    'Wednesday'
    'Thursday',
    'Friday',
    'Saturday',
    'Sunday'
);

ALTER TABLE scheduled_orders_days ADD COLUMN weekday weekdays;

commit;
