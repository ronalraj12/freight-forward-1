begin;
CREATE TYPE address_type AS ENUM (
    'home',
    'work',
    'other'
    );

CREATE TABLE address
(
    id           SERIAL PRIMARY KEY,
    user_id      INT REFERENCES users (id),
    address_data TEXT          NOT NULL,
    lat          DECIMAL(9, 6) NOT NULL,
    long         DECIMAL(9, 6) NOT NULL,
    type         address_type DEFAULT 'home'::address_type,
    created_at   TIMESTAMP    DEFAULT now(),
    updated_at   TIMESTAMP    DEFAULT now(),
    archived_at  TIMESTAMP,
    UNIQUE (user_id, type)
);
commit;