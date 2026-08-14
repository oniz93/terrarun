# Design Spec: Hybrid Land-Aware Seeding System

**Date:** 2026-05-25
**Status:** Approved
**Topic:** Implementing a dynamic, land-aware territory seeding system using Mapbox and PostGIS caching.

## 1. Goal
Provide a way to dynamically seed hexagonal territories (H3 Resolution 11) only on land, including support for coastlines and rivers, while minimizing Mapbox API costs and optimizing backend performance.

## 2. Architecture
The system follows a "Hybrid Cache" model where the local PostGIS database acts as a learned landmask.

### 2.1 Components
*   **Discovery Engine (Go Backend):** Responsible for coordinating the seeding logic, checking the local cache, and fetching new data from Mapbox.
*   **Land Cache (PostGIS):** Stores high-resolution vector geometry (Polygons) for landmasses, indexed by H3 Resolution 6 parent cells (~3.5km wide).
*   **H3 Library:** Used for coordinate-to-hex and parent-child relationship calculations.
*   **Mapbox Tilequery API:** Acts as the external oracle for geographic feature data.

## 3. Data Flow
1.  **Request:** The backend identifies a need to seed a territory at H3 index `H11`.
2.  **Parent Mapping:** The system calculates the Resolution 6 parent index: `P6 = h3.Cell(H11).Parent(6)`.
3.  **Local Cache Lookup:** 
    *   Query `land_cache` table by `P6`.
    *   If a geometry exists, proceed to Step 5.
4.  **Remote Discovery (Cache Miss):**
    *   Call Mapbox Tilequery API for the geographic center of `P6`.
    *   Retrieve vector land polygons within the tile.
    *   Parse the GeoJSON into a PostGIS `MultiPolygon`.
    *   Insert the `P6` index and the geometry into `land_cache`.
5.  **Spatial Precision Check:**
    *   Calculate the center point of `H11`.
    *   Perform a PostGIS `ST_Intersects` check between the cached geometry and the `H11` center point.
6.  **Final Decision:**
    *   If **Intersects**: Create the hex in the `hexes` table.
    *   If **No Intersection**: Skip the hex (it is water).

## 4. Database Schema

```sql
-- Enables spatial capabilities
CREATE EXTENSION IF NOT EXISTS postgis;

-- Stores the learned landmask in chunks
CREATE TABLE land_cache (
    h3_parent_index BIGINT PRIMARY KEY, -- The Res 6 "Discovery Bucket"
    geom GEOMETRY(MultiPolygon, 4326),  -- High-res land geometry from Mapbox
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for fast spatial lookups inside a cached chunk
CREATE INDEX idx_land_cache_geom ON land_cache USING GIST(geom);
```

## 5. Implementation Strategy
1.  **Backend Integration:** Create a Mapbox client in Go to handle Tilequery requests.
2.  **Migration:** Add the `land_cache` table and enable PostGIS.
3.  **H3 Utility Update:** Add parent-finding and point-checking wrappers.
4.  **Seeding Logic:** Implement the `DiscoverOrFetch` flow to replace existing "dumb" seeding.

## 6. Success Criteria
*   New hexes are never seeded in the Strait of Georgia or Burrard Inlet.
*   Rivers (e.g., Fraser River) are correctly represented as water gaps.
*   API calls to Mapbox do not exceed 1,000 for a 100km radius around Vancouver.
*   Territory rendering remains performant on the client device.
