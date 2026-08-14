CREATE TABLE seasons (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    start_date      TIMESTAMPTZ NOT NULL,
    end_date        TIMESTAMPTZ NOT NULL,
    tiers_config    JSONB NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_active UNIQUE (is_active)
        DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE season_participants (
    user_id         UUID NOT NULL REFERENCES users(id),
    season_id       UUID NOT NULL REFERENCES seasons(id),
    runner_points   INTEGER NOT NULL DEFAULT 0,
    territory_points INTEGER NOT NULL DEFAULT 0,
    current_tier    TEXT NOT NULL DEFAULT 'rookie',
    final_tier      TEXT,
    badge_awarded   BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (user_id, season_id)
);
