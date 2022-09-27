ALTER TABLE users
    ADD COLUMN flags        int         DEFAULT 0,
    ADD COLUMN unflagged_at timestamptz DEFAULT NOW();
