package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrarun/backend/pkg/h3util"
)

func main() {
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

	// Full Vancouver bbox provided by user
	minLat, minLng := 48.847295299126614, -123.3373648065106
	maxLat, maxLng := 49.4881382466163, -122.8215067934904
	res := 11

	cells, err := h3util.BBoxToCells(minLat, minLng, maxLat, maxLng, res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "BBox error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("BBox covers approximately %d cells at resolution %d.\n", len(cells), res)
	fmt.Printf("Seeding 40%% coverage...\n")

	count := 0
	// Using a smaller sample if it's too huge, but let's try to do it all in batches.
	for i, cell := range cells {
		// 40% coverage
		if rand.Float64() > 0.4 {
			continue
		}

		faction := "neon"
		if rand.Float64() > 0.5 {
			faction = "umbra"
		}

		hp := rand.Intn(5) + 1
		cLat, cLng, _ := h3util.CellToLatLng(cell)

		_, err = pool.Exec(ctx,
			`INSERT INTO hexes (h3_index, owned_by, hp, captured_at, last_decayed_at, lat, lng)
			 VALUES ($1, $2, $3, NOW(), NOW(), $4, $5)
			 ON CONFLICT (h3_index) DO NOTHING`,
			int64(cell), faction, hp, cLat, cLng)
		if err == nil {
			count++
		}

		if i > 0 && i%50000 == 0 {
			fmt.Printf("Processed %d cells, seeded %d so far...\n", i, count)
		}
	}

	fmt.Printf("Successfully seeded %d hexes in the full Vancouver area.\n", count)
}
