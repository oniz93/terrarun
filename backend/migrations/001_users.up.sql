CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    password_hash   TEXT,
    display_name    TEXT NOT NULL,
    phone_hash      TEXT,
    avatar_url      TEXT,
    faction         TEXT CHECK (faction IN ('neon', 'umbra')),
    account_level   INTEGER NOT NULL DEFAULT 1,
    account_xp      BIGINT NOT NULL DEFAULT 0,
    google_id       TEXT UNIQUE,
    apple_id        TEXT UNIQUE,
    runner_tier     TEXT NOT NULL DEFAULT 'free'
        CHECK (runner_tier IN ('free', 'runner', 'captain')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_faction ON users(faction) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_phone_hash ON users(phone_hash) WHERE deleted_at IS NULL AND phone_hash IS NOT NULL;
CREATE INDEX idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_users_apple_id ON users(apple_id) WHERE apple_id IS NOT NULL AND deleted_at IS NULL;
