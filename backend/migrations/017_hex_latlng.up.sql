-- Materialized lat/lng for fast bbox queries in territory/repository.go.
-- Populated by cmd/migrate_latlng (or during capture/seeding writes).
ALTER TABLE hexes ADD COLUMN IF NOT EXISTS lat DOUBLE PRECISION;
ALTER TABLE hexes ADD COLUMN IF NOT EXISTS lng DOUBLE PRECISION;

CREATE INDEX IF NOT EXISTS idx_hexes_latlng ON hexes(lat, lng);
