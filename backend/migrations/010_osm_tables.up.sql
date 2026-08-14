-- OSM-derived feature tables for land/water/forest/mountain classification
-- Used to replace Mapbox Tilequery API with local data

-- Global landmask (Natural Earth 10m land polygons)
CREATE TABLE IF NOT EXISTS global_landmask (
    id BIGSERIAL PRIMARY KEY,
    geom GEOMETRY(MultiPolygon, 4326)
);
CREATE INDEX IF NOT EXISTS idx_global_landmask_geom ON global_landmask USING GIST(geom);

-- Water bodies: lakes, rivers, reservoirs, wetlands, coastline
CREATE TABLE IF NOT EXISTS osm_water (
    id BIGSERIAL PRIMARY KEY,
    geom GEOMETRY(Geometry, 4326),
    class TEXT,  -- 'water', 'reservoir', 'riverbank', 'wetland', 'coastline'
    name TEXT
);
CREATE INDEX IF NOT EXISTS idx_osm_water_geom ON osm_water USING GIST(geom);

-- Forest and woodland areas
CREATE TABLE IF NOT EXISTS osm_forest (
    id BIGSERIAL PRIMARY KEY,
    geom GEOMETRY(Geometry, 4326),
    class TEXT  -- 'forest', 'wood', 'scrub'
);
CREATE INDEX IF NOT EXISTS idx_osm_forest_geom ON osm_forest USING GIST(geom);

-- Protected areas: national parks, nature reserves, etc.
CREATE TABLE IF NOT EXISTS osm_protected (
    id BIGSERIAL PRIMARY KEY,
    geom GEOMETRY(Geometry, 4326),
    class TEXT,  -- 'national_park', 'nature_reserve', 'protected_area'
    name TEXT
);
CREATE INDEX IF NOT EXISTS idx_osm_protected_geom ON osm_protected USING GIST(geom);

-- Mountain/peak features
CREATE TABLE IF NOT EXISTS osm_mountain (
    id BIGSERIAL PRIMARY KEY,
    geom GEOMETRY(Geometry, 4326),
    class TEXT,  -- 'peak', 'bare_rock', 'scree', 'ridge', 'glacier'
    name TEXT,
    elevation INTEGER
);
CREATE INDEX IF NOT EXISTS idx_osm_mountain_geom ON osm_mountain USING GIST(geom);
