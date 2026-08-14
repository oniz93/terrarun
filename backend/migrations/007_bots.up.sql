CREATE TABLE bots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name    TEXT NOT NULL,
    faction         TEXT NOT NULL CHECK (faction IN ('neon', 'umbra')),
    avatar_seed     TEXT,
    home_lat        DOUBLE PRECISION NOT NULL,
    home_lng        DOUBLE PRECISION NOT NULL,
    active_radius_km DOUBLE PRECISION NOT NULL DEFAULT 5,
    avg_distance_m  DOUBLE PRECISION NOT NULL DEFAULT 5000,
    total_runs      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at  TIMESTAMPTZ
);

CREATE INDEX idx_bots_home ON bots(home_lat, home_lng);
CREATE INDEX idx_bots_faction ON bots(faction, deactivated_at);
