-- In-app notifications referenced by internal/notification/service.go
CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    data        JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read        BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at DESC);
