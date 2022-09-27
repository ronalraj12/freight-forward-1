create table disputed_orders
(
    id          serial,
    order_id    int       not null,
    disputed_at timestamp not null,
    disputed_by int       not null,
    resolved_at int,
    resolved_by int
);