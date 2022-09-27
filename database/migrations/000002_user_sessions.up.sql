CREATE TABLE user_session
(
    id           SERIAL primary key,
    user_id      INT REFERENCES users (id) NOT NULL,
    token        TEXT                      NOT NULL,
    created_at   TIMESTAMP DEFAULT now(),
    last_used_at TIMESTAMP
);