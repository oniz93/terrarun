package main
import (
	"context"
	"fmt"
	"os"
	"math/rand"
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
	lat := 49.22536257164785
	lng := -123.00507100553435
	res := 11
	centerCell, _ := h3util.LatLngToCell(lat, lng, res)
	k := h3util.DiskDistance(res, 5.0)
	cells, _ := h3util.GridDisk(centerCell, k)
	count := 0
	for _, cell := range cells {
		faction := "neon"
		if count%2 == 0 {
			faction = "umbra"
		}
		cLat, cLng, _ := h3util.CellToLatLng(cell)
		_, err = pool.Exec(ctx,
			,
			int64(cell), faction, cLat, cLng)
		if err == nil {
			count++
		}
	}
	fmt.Printf("Successfully seeded %d hexes in Burnaby.\n", count)
}
