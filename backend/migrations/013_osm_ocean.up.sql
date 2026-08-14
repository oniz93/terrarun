CREATE TABLE IF NOT EXISTS osm_ocean (
    id BIGSERIAL PRIMARY KEY,
    geom GEOMETRY(MultiPolygon, 4326)
);
CREATE INDEX IF NOT EXISTS idx_osm_ocean_geom ON osm_ocean USING GIST(geom);
