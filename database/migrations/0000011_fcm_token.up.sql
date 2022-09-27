CREATE TABLE fcm_token(
    user_id INT PRIMARY KEY REFERENCES users(id),
    token TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP
);