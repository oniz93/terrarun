package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"
	"hash/fnv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrarun/backend/pkg/h3util"
	"github.com/uber/h3-go/v4"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()
	
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://terrarun:terrarun_dev@postgres:5432/terrarun?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	res := 11
	biomeRes := 6
	centerLat, centerLng := 49.2253, -123.0050
	center, _ := h3util.LatLngToCell(centerLat, centerLng, res)
	
	// 100km radius (~2000 hexes distance)
	k := 2000
	cells, _ := h3util.GridDisk(center, k)

	fmt.Printf("Starting High-Precision Water-Exclusion seeding for %d hexes...\n", len(cells))

	count := 0
	skipCount := 0
	limit := 1000000 

	for i, cell := range cells {
		// 40% density
		if rand.Float64() > 0.4 {
			continue
		}

		lat, lng, _ := h3util.CellToLatLng(cell)
		
		// 1. FAST Global Landmask Check
		var onLand bool
		err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM global_landmask WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))", lng, lat).Scan(&onLand)
		if err != nil || !onLand {
			continue
		}

		// 2. HIGH PRECISION OSM Water Exclusion
		var isWater bool
		err = pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM osm_water WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))", lng, lat).Scan(&isWater)
		if err != nil || isWater {
			skipCount++
			continue
		}

		parent, _ := h3.Cell(cell).Parent(biomeRes)
		h := fnv.New32a()
		h.Write([]byte(fmt.Sprintf("%d", int64(parent))))
		faction := "neon"
		if h.Sum32() % 2 == 0 {
			faction = "umbra"
		}

		hp := rand.Intn(10) + 1
		
		_, err = pool.Exec(ctx,
			"INSERT INTO hexes (h3_index, owned_by, hp, captured_at, last_decayed_at, is_natural, lat, lng) VALUES ($1, $2, $3, NOW(), NOW(), true, $4, $5) ON CONFLICT (h3_index) DO NOTHING",
			int64(cell), faction, hp, lat, lng)
		
		if err == nil {
			count++
		}

		if i % 1000 == 0 {
			fmt.Printf("Processed %d candidates... (Seeded: %d, Skipped/Water: %d)\n", i, count, skipCount)
		}

		if count >= limit {
			break
		}
	}

	fmt.Printf("Seeding complete. Total hexes seeded on land: %d\n", count)
}
