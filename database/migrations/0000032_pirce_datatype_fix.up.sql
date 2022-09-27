begin;

alter table items
    alter column price type decimal(20, 3);


alter table order_items
    alter column price type decimal(20, 3);

alter table scheduled_ordered_items
    alter column price type decimal(20, 3);

alter table orders
    alter column amount type decimal(20, 3);

commit;