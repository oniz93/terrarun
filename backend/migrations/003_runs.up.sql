CREATE TABLE runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'completed', 'flagged', 'rejected')),
    distance_m      DOUBLE PRECISION,
    elevation_gain_m DOUBLE PRECISION,
    avg_pace_s_per_km DOUBLE PRECISION,
    max_speed_kmh   DOUBLE PRECISION,
    avg_hr_bpm      INTEGER,
    max_hr_bpm      INTEGER,
    calories        INTEGER,
    closed_loop     BOOLEAN,
    capture_mode    TEXT CHECK (capture_mode IN ('polygon', 'path')),
    loop_snap_distance_m DOUBLE PRECISION,
    path_buffer_radius_m DOUBLE PRECISION,
    polyline        TEXT,
    territory_points INTEGER NOT NULL DEFAULT 0,
    runner_points   INTEGER NOT NULL DEFAULT 0,
    social_run      BOOLEAN NOT NULL DEFAULT FALSE,
    social_leader   UUID REFERENCES users(id),
    social_participants UUID[],
    health_source   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_runs_user ON runs(user_id, created_at DESC);
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_start_time ON runs(start_time);

CREATE TABLE gps_points (
    run_id          UUID NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL,
    lat             DOUBLE PRECISION NOT NULL,
    lng             DOUBLE PRECISION NOT NULL,
    altitude        DOUBLE PRECISION,
    speed           DOUBLE PRECISION,
    horizontal_accuracy DOUBLE PRECISION,
    heart_rate      INTEGER
);

SELECT create_hypertable('gps_points', 'timestamp', if_not_exists => TRUE);
CREATE INDEX idx_gps_points_run ON gps_points(run_id, timestamp);
