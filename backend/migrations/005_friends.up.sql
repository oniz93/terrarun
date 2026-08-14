CREATE TABLE friends (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'blocked')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, friend_id),
    CHECK (user_id != friend_id)
);

CREATE INDEX idx_friends_friend ON friends(friend_id, status);
