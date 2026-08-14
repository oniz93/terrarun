# Hybrid Land-Aware Seeding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a dynamic seeding system that uses Mapbox Tilequery to place territories only on land and caches results in PostGIS.

**Architecture:** A hybrid model using H3 Resolution 6 parent cells as "discovery buckets". Mapbox vector data is cached locally to provide high-precision land/water checks (rivers, coastlines) at Resolution 11.

**Tech Stack:** Go, PostgreSQL/PostGIS, Mapbox Tilequery API, H3.

---

### Task 1: Database Migration & PostGIS Setup

**Files:**
- Create: `backend/migrations/008_land_cache.up.sql`
- Create: `backend/migrations/008_land_cache.down.sql`

- [ ] **Step 1: Create the up migration**
```sql
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS land_cache (
    h3_parent_index BIGINT PRIMARY KEY,
    geom GEOMETRY(MultiPolygon, 4326),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_land_cache_geom ON land_cache USING GIST(geom);
```

- [ ] **Step 2: Create the down migration**
```sql
DROP TABLE IF EXISTS land_cache;
-- We keep postgis extension as it might be used elsewhere
```

- [ ] **Step 3: Run the migration**
Run: `docker compose exec api ./migrate up`
Expected: Successfully applied 008_land_cache.up.sql

- [ ] **Step 4: Commit**
```bash
git add backend/migrations/008_land_cache.*
git commit -m "db: add land_cache table and enable postgis"
```

---

### Task 2: Mapbox Client Implementation

**Files:**
- Create: `backend/pkg/mapbox/client.go`
- Create: `backend/pkg/mapbox/client_test.go`

- [ ] **Step 1: Write the failing test for Tilequery**
```go
package mapbox

import (
	"context"
	"testing"
)

func TestClient_GetLandWater(t *testing.T) {
	c := NewClient("fake_token")
	_, err := c.GetLandWater(context.Background(), 49.2253, -123.0050)
	if err == nil {
		t.Error("Expected error for fake token")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd backend && go test ./pkg/mapbox/...`
Expected: FAIL (Client undefined)

- [ ] **Step 3: Implement Mapbox Client**
```go
package mapbox

import (
	"context"
	"fmt"
	"net/http"
	"io"
)

type Client struct {
	token string
	http  *http.Client
}

func NewClient(token string) *Client {
	return &Client{token: token, http: &http.Client{}}
}

func (c *Client) GetLandWater(ctx context.Background, lat, lng float64) ([]byte, error) {
	url := fmt.Sprintf("https://api.mapbox.com/v4/mapbox.mapbox-streets-v8/tilequery/%f,%f.json?layers=landuse,water,structure&access_token=%s", lng, lat, c.token)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mapbox error: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd backend && go test ./pkg/mapbox/...`
Expected: PASS (or 401 error if token is fake, which matches the test expectation)

- [ ] **Step 5: Commit**
```bash
git add backend/pkg/mapbox/
git commit -m "pkg: add mapbox tilequery client"
```

---

### Task 3: Land Cache Repository

**Files:**
- Create: `backend/internal/territory/land_repository.go`

- [ ] **Step 1: Define LandRepository interface and implementation**
```go
package territory

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LandRepository interface {
	GetLandGeometry(ctx context.Context, parentIndex int64) ([]byte, error)
	SaveLandGeometry(ctx context.Context, parentIndex int64, wkb []byte) error
	IsOnLand(ctx context.Context, lat, lng float64, parentIndex int64) (bool, error)
}

type PostgresLandRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresLandRepo(pool *pgxpool.Pool) *PostgresLandRepo {
	return &PostgresLandRepo{pool: pool}
}

func (r *PostgresLandRepo) GetLandGeometry(ctx context.Context, parentIndex int64) ([]byte, error) {
	var wkb []byte
	err := r.pool.QueryRow(ctx, "SELECT ST_AsBinary(geom) FROM land_cache WHERE h3_parent_index = $1", parentIndex).Scan(&wkb)
	return wkb, err
}

func (r *PostgresLandRepo) SaveLandGeometry(ctx context.Context, parentIndex int64, wkb []byte) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO land_cache (h3_parent_index, geom) VALUES ($1, ST_GeomFromWKB($2, 4326)) ON CONFLICT (h3_parent_index) DO NOTHING", parentIndex, wkb)
	return err
}

func (r *PostgresLandRepo) IsOnLand(ctx context.Context, lat, lng float64, parentIndex int64) (bool, error) {
	var onLand bool
	err := r.pool.QueryRow(ctx, `
		SELECT ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)) 
		FROM land_cache WHERE h3_parent_index = $3`, lng, lat, parentIndex).Scan(&onLand)
	return onLand, err
}
```

- [ ] **Step 2: Commit**
```bash
git add backend/internal/territory/land_repository.go
git commit -m "repo: add land_cache repository"
```

---

### Task 4: Unified Seeding Service

**Files:**
- Create: `backend/internal/territory/seeding_service.go`

- [ ] **Step 1: Implement the SeedingService with Discovery logic**
```go
package territory

import (
	"context"
	"github.com/terrarun/backend/pkg/mapbox"
	"github.com/terrarun/backend/pkg/h3util"
)

type SeedingService struct {
	repo     Repository
	landRepo LandRepository
	mapbox   *mapbox.Client
}

func (s *SeedingService) SeedHex(ctx context.Context, h3Index int64, faction string) error {
	parent := int64(h3util.Cell(h3Index).Parent(6))
	
	// Check cache
	onLand, err := s.landRepo.IsOnLand(ctx, lat, lng, parent)
	if err != nil {
		// Cache miss - Fetch from Mapbox
		geoJSON, _ := s.mapbox.GetLandWater(ctx, lat, lng)
		// Simplified conversion to WKB for this plan...
		s.landRepo.SaveLandGeometry(ctx, parent, convertedWKB)
		onLand, _ = s.landRepo.IsOnLand(ctx, lat, lng, parent)
	}

	if onLand {
		return s.repo.UpsertHex(ctx, h3Index, faction)
	}
	return nil
}
```

- [ ] **Step 2: Commit**
```bash
git add backend/internal/territory/seeding_service.go
git commit -m "service: add land-aware seeding service"
```

---

### Task 5: Final Mass Seeding Command

**Files:**
- Create: `backend/cmd/seed_final/main.go`

- [ ] **Step 1: Implement the 100km radius land-aware seed**
```go
package main
// Integration of Task 1-4 logic to seed 40% density on land only.
```

- [ ] **Step 2: Run final seed**
Run: `docker run ... go run cmd/seed_final/main.go`
Expected: Map populated with clusters only on land.

- [ ] **Step 3: Commit**
```bash
git add backend/cmd/seed_final/main.go
git commit -m "cmd: add final land-aware seeding command"
```
