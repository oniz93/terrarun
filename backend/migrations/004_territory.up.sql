CREATE TABLE hexes (
    h3_index        BIGINT PRIMARY KEY,
    owned_by        TEXT CHECK (owned_by IN ('neon', 'umbra')),
    hp              SMALLINT NOT NULL DEFAULT 1
        CHECK (hp >= 1 AND hp <= 10),
    captured_by     UUID REFERENCES runs(id),
    captured_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_decayed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_natural      BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_hexes_owned ON hexes(owned_by) WHERE owned_by IS NOT NULL;
CREATE INDEX idx_hexes_decay ON hexes(last_decayed_at) WHERE owned_by IS NOT NULL;

CREATE TABLE territory_changes (
    id              BIGSERIAL PRIMARY KEY,
    h3_index        BIGINT NOT NULL,
    run_id          UUID REFERENCES runs(id),
    previous_owner  TEXT,
    new_owner       TEXT CHECK (new_owner IN ('neon', 'umbra')),
    hp_before       SMALLINT NOT NULL,
    hp_after        SMALLINT NOT NULL,
    changed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_territory_changes_time ON territory_changes(changed_at);
CREATE INDEX idx_territory_changes_run ON territory_changes(run_id);
