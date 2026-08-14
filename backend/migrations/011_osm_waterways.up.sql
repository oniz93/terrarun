-- Linear water features (rivers, streams, canals) from OSM waterways shapefile
-- These are LineString geometries (not polygons) representing river centerlines
-- Used to exclude hexes that intersect narrow rivers not present in osm_water polygons

CREATE TABLE IF NOT EXISTS osm_waterways (
    id BIGSERIAL PRIMARY KEY,
    geom GEOMETRY(LineString, 4326),
    class TEXT,  -- 'river', 'stream', 'canal', 'drain', 'ditch'
    name TEXT
);
CREATE INDEX IF NOT EXISTS idx_osm_waterways_geom ON osm_waterways USING GIST(geom);
