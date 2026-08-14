package main

import (
	"context"
	"fmt"
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

	lat := 49.034565721377476
	lng := -123.16559626696728
	radiusKM := 5.0
	res := 11

	centerCell, err := h3util.LatLngToCell(lat, lng, res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "H3 error: %v\n", err)
		os.Exit(1)
	}

	k := h3util.DiskDistance(res, radiusKM)
	cells, err := h3util.GridDisk(centerCell, k)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GridDisk error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Seeding %d cells with 100%% coverage around (%f, %f)...\n", len(cells), lat, lng)

	count := 0
	for _, cell := range cells {
		faction := "neon"
		if count%2 == 0 {
			faction = "umbra"
		}
		
		lat, lng, _ := h3util.CellToLatLng(cell)

		_, err = pool.Exec(ctx,
			`INSERT INTO hexes (h3_index, owned_by, hp, captured_at, last_decayed_at, lat, lng)
			 VALUES ($1, $2, 10, NOW(), NOW(), $3, $4)
			 ON CONFLICT (h3_index) DO UPDATE SET owned_by = $2, hp = 10`,
			int64(cell), faction, lat, lng)
		if err == nil {
			count++
		}
	}

	fmt.Printf("Successfully seeded %d hexes.\n", count)
}
