BEGIN;

CREATE TYPE image_type AS ENUM (
    'profile'
    );

CREATE TABLE images
(
    id          SERIAL PRIMARY KEY,
    type        image_type NOT NULL,
    bucket      TEXT,
    path        TEXT,
    created_at  TIMESTAMP DEFAULT now(),
    archived_at TIMESTAMP,
    updated_at  TIMESTAMP
);

CREATE TABLE users
(
    id            SERIAL PRIMARY KEY,
    name          TEXT,
    phone         TEXT NOT NULL,
    email         TEXT,
    profile_image INT REFERENCES images (id),
    created_at    TIMESTAMP DEFAULT now(),
    updated_at    TIMESTAMP,
    archived_at   TIMESTAMP
);

CREATE TYPE permission AS ENUM (
    'user',
    'cart-boy',
    'delivery-boy',
    'store-manager',
    'admin'
    );

CREATE TABLE user_permission
(
    user_id         int        NOT NULL REFERENCES users (id),
    permission_type permission NOT NULL DEFAULT 'user'::permission,
    UNIQUE (user_id, permission_type)
);
COMMIT;