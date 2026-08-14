package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OverpassResponse struct {
	Elements []struct {
		Type     string `json:"type"`
		Geometry []struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lon"`
		} `json:"geometry"`
	} `json:"elements"`
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

	data, err := os.ReadFile("vancouver_water.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read file: %v\n", err)
		os.Exit(1)
	}

	var or OverpassResponse
	if err := json.Unmarshal(data, &or); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to unmarshal: %v\n", err)
		os.Exit(1)
	}

	pool.Exec(ctx, "CREATE TABLE IF NOT EXISTS osm_water (id SERIAL PRIMARY KEY, geom GEOMETRY(Geometry, 4326))")
	pool.Exec(ctx, "TRUNCATE TABLE osm_water")

	fmt.Printf("Loading %d water features...\n", len(or.Elements))

	for i, el := range or.Elements {
		if len(el.Geometry) < 3 {
			continue
		}

		var points []string
		for _, p := range el.Geometry {
			points = append(points, fmt.Sprintf("%f %f", p.Lng, p.Lat))
		}
		// Close the loop if needed
		if el.Geometry[0] != el.Geometry[len(el.Geometry)-1] {
			points = append(points, fmt.Sprintf("%f %f", el.Geometry[0].Lng, el.Geometry[0].Lat))
		}

		wkt := fmt.Sprintf("POLYGON((%s))", strings.Join(points, ","))
		
		_, err = pool.Exec(ctx, "INSERT INTO osm_water (geom) VALUES (ST_MakeValid(ST_GeomFromText($1, 4326)))", wkt)
		if err != nil {
			// Some might be invalid, we just skip or log
			continue
		}

		if i % 1000 == 0 {
			fmt.Printf("Loaded %d...\n", i)
		}
	}

	fmt.Printf("Done. Creating spatial index...\n")
	pool.Exec(ctx, "CREATE INDEX IF NOT EXISTS idx_osm_water_geom ON osm_water USING GIST(geom)")
	fmt.Printf("Finished.\n")
}
