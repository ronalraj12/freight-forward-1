BEGIN;

CREATE TABLE categories
(
    id          SERIAL PRIMARY KEY,
    category    text,
    created_at  TIMESTAMP DEFAULT now(),
    archived_at TIMESTAMP,
    updated_at  TIMESTAMP
);

CREATE TABLE items
(
    id          SERIAL PRIMARY KEY,
    name        text,
    price       DECIMAL(5, 3),
    in_stock    boolean,
    category    int references categories (id),
    created_at  TIMESTAMP DEFAULT now(),

    archived_at TIMESTAMP,
    updated_at  TIMESTAMP
);

COMMIT;