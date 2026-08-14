CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS land_cache (
    h3_parent_index BIGINT PRIMARY KEY,
    geom GEOMETRY(MultiPolygon, 4326),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_land_cache_geom ON land_cache USING GIST(geom);
