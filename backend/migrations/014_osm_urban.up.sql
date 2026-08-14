CREATE TABLE IF NOT EXISTS osm_urban (
    id BIGSERIAL PRIMARY KEY,
    geom GEOMETRY(Geometry, 4326),
    class TEXT
);
CREATE INDEX IF NOT EXISTS idx_osm_urban_geom ON osm_urban USING GIST(geom);
