-- Subdivided landmask for faster spatial joins
-- Natural Earth 10m land polygons are huge (entire continents in ~20 polygons)
-- ST_Subdivide breaks them into ~20k smaller polygons for faster ST_Intersects

DROP TABLE IF EXISTS global_landmask_subdivided;
CREATE TABLE global_landmask_subdivided AS SELECT ST_Subdivide(geom, 256) as geom FROM global_landmask;
CREATE INDEX idx_landmask_sub_geom ON global_landmask_subdivided USING GIST(geom);
