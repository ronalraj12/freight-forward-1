begin;

Alter table orders
    drop column order_otp;


create table order_otp
(
    order_id int not null references orders (id),
    otp      int not null,
    unique (order_id, otp)
);

commit;