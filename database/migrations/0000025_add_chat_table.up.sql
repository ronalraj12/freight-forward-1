create table chat
(
    Id         serial,
    order_id   int,
    sender     int,
    receiver   int,
    message    text,
    created_at timestamp
)