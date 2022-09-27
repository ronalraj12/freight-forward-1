begin;

ALTER TABLE orders
    ALTER COLUMN user_rating TYPE decimal;
Alter table orders
    alter column staff_rating type decimal;

commit;