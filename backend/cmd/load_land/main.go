package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FeatureCollection struct {
	Features []struct {
		Geometry json.RawMessage `json:"geometry"`
	} `json:"features"`
}

func main() {
	ctx := context.Background()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://terrarun:terrarun_dev@postgres:5432/terrarun?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	data, err := os.ReadFile("land.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read file: %v\n", err)
		os.Exit(1)
	}

	var fc FeatureCollection
	if err := json.Unmarshal(data, &fc); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to unmarshal: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loading %d features...\n", len(fc.Features))

	for i, f := range fc.Features {
		_, err = pool.Exec(ctx, "INSERT INTO global_landmask (geom) VALUES (ST_Multi(ST_GeomFromGeoJSON($1)))", string(f.Geometry))
		if err != nil {
			fmt.Printf("Error inserting feature %d: %v\n", i, err)
		}
		if i % 500 == 0 {
			fmt.Printf("Loaded %d...\n", i)
		}
	}

	fmt.Printf("Done. Creating spatial index...\n")
	pool.Exec(ctx, "CREATE INDEX IF NOT EXISTS idx_global_landmask_geom ON global_landmask USING GIST(geom)")
	fmt.Printf("Finished.\n")
}
