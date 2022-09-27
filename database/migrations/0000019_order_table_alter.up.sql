begin;

alter table orders
    drop column items;

create table order_items
(
    order_id int references orders(id),
    name          text,
    price         numeric(5, 3),
    category      text,
    base_quantity text,
    quantity      int,
    bucket        text,
    path          text
);



commit;