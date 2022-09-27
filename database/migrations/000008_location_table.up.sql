CREATE TABLE location
(
    staff_id   int primary key references users (id),
    lat        decimal(9, 6),
    long       decimal(9, 6),
    updated_at timestamp
);