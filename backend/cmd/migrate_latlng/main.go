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

	rows, err := pool.Query(ctx, "SELECT h3_index FROM hexes WHERE lat IS NULL")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query error: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	type hexPos struct {
		id  int64
		lat float64
		lng float64
	}

	var batch []hexPos
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		lat, lng, err := h3util.CellToLatLng(h3util.Int64ToH3Index(id))
		if err != nil {
			continue
		}
		batch = append(batch, hexPos{id, lat, lng})
	}

	fmt.Printf("Updating %d hexes...\n", len(batch))

	for _, h := range batch {
		_, err = pool.Exec(ctx, "UPDATE hexes SET lat = $1, lng = $2 WHERE h3_index = $3", h.lat, h.lng, h.id)
		if err != nil {
			fmt.Printf("Update error for %d: %v\n", h.id, err)
		}
	}

	fmt.Println("Done.")
}
