CREATE TABLE IF NOT EXISTS water_cache (
    h3_parent_index BIGINT PRIMARY KEY,
    geom GEOMETRY(MultiPolygon, 4326),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_water_cache_geom ON water_cache USING GIST(geom);
