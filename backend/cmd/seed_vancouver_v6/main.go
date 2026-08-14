package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ojrac/opensimplex-go"
	"github.com/terrarun/backend/pkg/h3util"
	"github.com/uber/h3-go/v4"
)

const parentRes = 9

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://terrarun:terrarun_dev@localhost:5432/terrarun?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fatal("db connect: %v", err)
	}
	defer pool.Close()

	centerLat, centerLng := 49.2253, -123.0050
	center, _ := h3util.LatLngToCell(centerLat, centerLng, 11)
	radiusKM := 30.0
	k := h3util.DiskDistance(11, radiusKM)
	cells, _ := h3util.GridDisk(center, k)

	fmt.Printf("=== TerraRun Vancouver Seeder v6 ===\n")
	fmt.Printf("Center: (%.4f, %.4f), Radius: %.0fkm\n", centerLat, centerLng, radiusKM)
	fmt.Printf("Res 11 cells in disk: %d\n\n", len(cells))

	totalStart := time.Now()

	fmt.Println("Step 1: Generate candidates + hex polygon WKT + extract unique Res 9 parents...")
	candidates, parents := prepareCandidates(cells)
	fmt.Printf("  Candidates (Res 11, 40%%): %d\n", len(candidates))
	fmt.Printf("  Unique Res 9 parents: %d\n\n", len(parents))

	fmt.Println("Step 2: Classify Res 9 parent cells (forest/mountain/protected)...")
	parentFeatures := classifyParents(ctx, dbURL, parents)

	fmt.Println("Step 3: COPY hex polygons to PostGIS + find excluded cells (water/ocean)...")
	excludedCells := findExcludedCells(ctx, dbURL, candidates)

	fmt.Println("Step 4: Find seedable cells (intersect osm_urban)...")
	seedableCells := findSeedableCells(ctx, dbURL, candidates)

	fmt.Println("Step 5: Generate territory field (simplex noise + growth + HP gradient)...")
	tf := generateTerritoryField(candidates, seedableCells)

	fmt.Println("Step 6: Seed using territory field + exclusion + urban-only...")
	stats := seedFromCache(ctx, dbURL, candidates, parentFeatures, excludedCells, seedableCells, tf)

	fmt.Println()
	fmt.Println("=== Results ===")
	fmt.Printf("Total time: %s\n", time.Since(totalStart).Round(time.Millisecond))
	fmt.Printf("\nHexes seeded: %d\n", stats.seeded)
	fmt.Printf("  Forest:     %d\n", stats.forest)
	fmt.Printf("  Mountain:   %d\n", stats.mountain)
	fmt.Printf("  Protected:  %d\n", stats.protected)
	fmt.Printf("  Urban/other:%d\n", stats.seeded-stats.forest-stats.mountain-stats.protected)
	fmt.Printf("\nSkipped:\n")
	fmt.Printf("  Water+Ocean: %d\n", stats.excluded)
	fmt.Printf("  Non-urban:   %d\n", stats.nonUrban)
	fmt.Printf("  Neutral:     %d\n", stats.neutral)

	pool.Exec(ctx, "DROP TABLE IF EXISTS seed_candidates")
}

type resolve struct {
	h3     int64
	lat    float64
	lng    float64
	wkt    string
	parent int64
}

type features struct {
	onForest    bool
	onMountain  bool
	onProtected bool
}

type seedStats struct {
	seeded   int
	excluded int
	nonUrban int
	neutral  int
	forest   int
	mountain int
	protected int
}

type territoryField struct {
	faction map[int64]int8
	hp      map[int64]int
}

func prepareCandidates(cells []h3.Cell) ([]resolve, []resolve) {
	candidates := make([]resolve, 0, int(float64(len(cells))*0.4+1000))
	parentMap := make(map[int64]resolve)
	parentSeen := make(map[int64]bool)

	for _, cell := range cells {
		if rand.Float64() > 0.4 {
			continue
		}
		lat, lng, _ := h3util.CellToLatLng(cell)
		parent, _ := cell.Parent(parentRes)
		pid := int64(parent)

		wkt := hexToWKT(cell)

		candidates = append(candidates, resolve{
			h3:     int64(cell),
			lat:    lat,
			lng:    lng,
			wkt:    wkt,
			parent: pid,
		})

		if !parentSeen[pid] {
			parentSeen[pid] = true
			plat, plng, _ := h3util.CellToLatLng(parent)
			parentMap[pid] = resolve{h3: pid, lat: plat, lng: plng}
		}
	}

	parents := make([]resolve, 0, len(parentMap))
	for _, p := range parentMap {
		parents = append(parents, p)
	}
	return candidates, parents
}

func hexToWKT(cell h3.Cell) string {
	boundary, err := h3.CellToBoundary(cell)
	if err != nil || len(boundary) < 3 {
		return ""
	}

	var coords []string
	for _, ll := range boundary {
		coords = append(coords,
			strconv.FormatFloat(ll.Lng, 'f', 8, 64)+" "+
				strconv.FormatFloat(ll.Lat, 'f', 8, 64))
	}
	first := coords[0]
	coords = append(coords, first)
	return "POLYGON((" + strings.Join(coords, ",") + "))"
}

func classifyParents(ctx context.Context, dbURL string, parents []resolve) map[int64]features {
	result := make(map[int64]features, len(parents))
	var mu sync.Mutex
	var done atomic.Int64
	var failed atomic.Int64
	total := len(parents)

	workers := 8
	var wg sync.WaitGroup
	classifyStart := time.Now()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			wp, err := pgxpool.New(ctx, dbURL)
			if err != nil {
				failed.Add(int64(total))
				return
			}
			defer wp.Close()

			for i := workerID; i < total; i += workers {
				p := parents[i]
				var f features
				err := wp.QueryRow(ctx, classifyQuery, p.lng, p.lat).Scan(&f.onForest, &f.onMountain, &f.onProtected)
				if err != nil {
					failed.Add(1)
					continue
				}

				mu.Lock()
				result[p.h3] = f
				mu.Unlock()

				d := done.Add(1)
				if d%1000 == 0 || int(d) >= total {
					elapsed := time.Since(classifyStart)
					fmt.Printf("  [%s] %d/%d parents (%.0f/s)\n",
						elapsed.Round(time.Second), d, total, float64(d)/elapsed.Seconds())
				}
			}
		}(w)
	}
	wg.Wait()

	if f := failed.Load(); f > 0 {
		fmt.Printf("  WARNING: %d/%d parents failed\n", f, total)
	}
	return result
}

func findExcludedCells(ctx context.Context, dbURL string, candidates []resolve) map[int64]bool {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil
	}
	defer pool.Close()

	minLng, minLat, maxLng, maxLat := candidates[0].lng, candidates[0].lat, candidates[0].lng, candidates[0].lat
	for _, c := range candidates {
		if c.lng < minLng {
			minLng = c.lng
		}
		if c.lng > maxLng {
			maxLng = c.lng
		}
		if c.lat < minLat {
			minLat = c.lat
		}
		if c.lat > maxLat {
			maxLat = c.lat
		}
	}
	pad := 0.01
	minLng -= pad
	minLat -= pad
	maxLng += pad
	maxLat += pad
	fmt.Printf("  BBox: [%.4f, %.4f] to [%.4f, %.4f]\n", minLng, minLat, maxLng, maxLat)

	pool.Exec(ctx, "DROP TABLE IF EXISTS seed_candidates")
	pool.Exec(ctx, `CREATE TABLE seed_candidates (
		h3_index BIGINT PRIMARY KEY,
		wkt TEXT,
		geom GEOMETRY(Polygon, 4326)
	)`)

	copyStart := time.Now()
	copied, _ := pool.CopyFrom(ctx, pgx.Identifier{"seed_candidates"}, []string{"h3_index", "wkt"},
		pgx.CopyFromSlice(len(candidates), func(i int) ([]any, error) {
			return []any{candidates[i].h3, candidates[i].wkt}, nil
		}),
	)
	fmt.Printf("  COPY: %d rows in %s\n", copied, time.Since(copyStart).Round(time.Millisecond))

	convStart := time.Now()
	if _, err := pool.Exec(ctx, "UPDATE seed_candidates SET geom = ST_GeomFromText(wkt, 4326)"); err != nil {
		fmt.Fprintf(os.Stderr, "  geom conversion error: %v\n", err)
		return nil
	}
	fmt.Printf("  WKT->geom: %s\n", time.Since(convStart).Round(time.Millisecond))

	idxStart := time.Now()
	if _, err := pool.Exec(ctx, "CREATE INDEX idx_seed_candidates_geom ON seed_candidates USING GIST(geom)"); err != nil {
		fmt.Fprintf(os.Stderr, "  index error: %v\n", err)
		return nil
	}
	fmt.Printf("  GIST index: %s\n", time.Since(idxStart).Round(time.Millisecond))

	excluded := make(map[int64]bool)

	waterStart := time.Now()
	rows, err := pool.Query(ctx, `
		SELECT c.h3_index FROM seed_candidates c
		JOIN osm_water w ON w.geom && ST_MakeEnvelope($1,$2,$3,$4,4326)
		                  AND w.geom && c.geom
		                  AND ST_Intersects(w.geom, c.geom)
	`, minLng, minLat, maxLng, maxLat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  water query error: %v\n", err)
	} else {
		n := 0
		for rows.Next() {
			var h3 int64
			if err := rows.Scan(&h3); err != nil {
				continue
			}
			excluded[h3] = true
			n++
		}
		rows.Close()
		fmt.Printf("  Water (polygons):  %d cells in %s\n", n, time.Since(waterStart).Round(time.Millisecond))
	}

	wwStart := time.Now()
	rows, err = pool.Query(ctx, `
		SELECT c.h3_index FROM seed_candidates c
		JOIN osm_waterways w ON w.geom && ST_MakeEnvelope($1,$2,$3,$4,4326)
		                     AND w.geom && ST_Expand(c.geom, 0.0005)
		                     AND ST_DWithin(w.geom, c.geom, 0.0002)
	`, minLng, minLat, maxLng, maxLat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  waterways query error: %v\n", err)
	} else {
		n := 0
		for rows.Next() {
			var h3 int64
			if err := rows.Scan(&h3); err != nil {
				continue
			}
			if !excluded[h3] {
				excluded[h3] = true
				n++
			}
		}
		rows.Close()
		fmt.Printf("  Waterways (lines):  %d cells in %s\n", n, time.Since(wwStart).Round(time.Millisecond))
	}

	oceanStart := time.Now()
	rows, err = pool.Query(ctx, `
		SELECT c.h3_index FROM seed_candidates c
		JOIN osm_ocean o ON o.geom && ST_MakeEnvelope($1,$2,$3,$4,4326)
		                 AND o.geom && c.geom
		                 AND ST_Intersects(o.geom, c.geom)
	`, minLng, minLat, maxLng, maxLat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ocean query error: %v\n", err)
	} else {
		n := 0
		for rows.Next() {
			var h3 int64
			if err := rows.Scan(&h3); err != nil {
				continue
			}
			if !excluded[h3] {
				excluded[h3] = true
				n++
			}
		}
		rows.Close()
		fmt.Printf("  Ocean (OSM):        %d cells in %s\n", n, time.Since(oceanStart).Round(time.Millisecond))
	}

	fmt.Printf("  Total excluded: %d cells\n", len(excluded))
	return excluded
}

func findSeedableCells(ctx context.Context, dbURL string, candidates []resolve) map[int64]bool {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil
	}
	defer pool.Close()

	minLng, minLat, maxLng, maxLat := candidates[0].lng, candidates[0].lat, candidates[0].lng, candidates[0].lat
	for _, c := range candidates {
		if c.lng < minLng {
			minLng = c.lng
		}
		if c.lng > maxLng {
			maxLng = c.lng
		}
		if c.lat < minLat {
			minLat = c.lat
		}
		if c.lat > maxLat {
			maxLat = c.lat
		}
	}
	pad := 0.01
	minLng -= pad
	minLat -= pad
	maxLng += pad
	maxLat += pad

	urbanStart := time.Now()
	seedable := make(map[int64]bool)
	rows, err := pool.Query(ctx, `
		SELECT c.h3_index FROM seed_candidates c
		JOIN osm_urban u ON u.geom && ST_MakeEnvelope($1,$2,$3,$4,4326)
		                  AND u.geom && c.geom
		                  AND ST_Intersects(u.geom, c.geom)
	`, minLng, minLat, maxLng, maxLat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  urban query error: %v\n", err)
		return seedable
	}
	defer rows.Close()

	for rows.Next() {
		var h3 int64
		if err := rows.Scan(&h3); err != nil {
			continue
		}
		seedable[h3] = true
	}

	fmt.Printf("  Urban seedable:     %d cells in %s\n", len(seedable), time.Since(urbanStart).Round(time.Millisecond))
	return seedable
}

func generateTerritoryField(candidates []resolve, seedableCells map[int64]bool) *territoryField {
	start := time.Now()

	candMap := make(map[int64]resolve, len(candidates))
	for _, c := range candidates {
		candMap[c.h3] = c
	}

	factionNoise := opensimplex.New(42)
	claimNoise := opensimplex.New(137)

	const (
		freq1, freq2       = 80.0, 200.0
		amp1, amp2         = 1.0, 0.35
		claimFreq          = 18.0
		claimThreshold     = -0.55
	)

	faction := make(map[int64]int8, len(seedableCells))
	noiseVals := make(map[int64]float64, len(seedableCells))
	neutral := make(map[int64]bool, len(seedableCells))

	for h3 := range seedableCells {
		c, ok := candMap[h3]
		if !ok {
			continue
		}

		v1 := factionNoise.Eval2(c.lng*freq1, c.lat*freq1) * amp1
		v2 := factionNoise.Eval2(c.lng*freq2, c.lat*freq2) * amp2
		noiseVals[h3] = (v1 + v2) / (amp1 + amp2)

		claim := claimNoise.Eval2(c.lng*claimFreq, c.lat*claimFreq)
		if claim < claimThreshold {
			neutral[h3] = true
		}
	}

	claimedNoise := make([]float64, 0, len(noiseVals))
	for h3, nv := range noiseVals {
		if neutral[h3] {
			continue
		}
		claimedNoise = append(claimedNoise, nv)
	}
	sort.Float64s(claimedNoise)
	median := claimedNoise[len(claimedNoise)/2]

	for h3, nv := range noiseVals {
		if neutral[h3] {
			faction[h3] = 0
		} else if nv > median {
			faction[h3] = 1
		} else {
			faction[h3] = 2
		}
	}

	noiseStart := time.Since(start)
	neutralAfterNoise := 0
	for _, f := range faction {
		if f == 0 {
			neutralAfterNoise++
		}
	}
	fmt.Printf("  Noise base:         %d neon, %d umbra, %d neutral (%s)\n",
		countFaction(faction, 1), countFaction(faction, 2), neutralAfterNoise, noiseStart.Round(time.Millisecond))

	neighbors := buildNeighborGraph(seedableCells)
	fmt.Printf("  Neighbor graph:     %d entries (%s)\n", len(neighbors), time.Since(start).Round(time.Millisecond))

	for round := 0; round < 3; round++ {
		claimed := 0
		for h3 := range seedableCells {
			if faction[h3] != 0 {
				continue
			}
			neonN, umbraN := 0, 0
			for _, n := range neighbors[h3] {
				switch faction[n] {
				case 1:
					neonN++
				case 2:
					umbraN++
				}
			}
			if neonN >= 2 && neonN > umbraN {
				if rand.Float64() < 0.55 {
					faction[h3] = 1
					claimed++
				}
			} else if umbraN >= 2 && umbraN > neonN {
				if rand.Float64() < 0.55 {
					faction[h3] = 2
					claimed++
				}
			}
		}
		fmt.Printf("  Growth round %d:      +%d claimed\n", round+1, claimed)
	}

	flipped := 0
	for h3 := range seedableCells {
		f := faction[h3]
		if f == 0 {
			continue
		}
		enemyCount := 0
		enemyFaction := int8(0)
		for _, n := range neighbors[h3] {
			nf := faction[n]
			if nf != 0 && nf != f {
				enemyCount++
				enemyFaction = nf
			}
		}
		if enemyCount >= 1 && rand.Float64() < 0.04 {
			faction[h3] = enemyFaction
			flipped++
		}
	}
	fmt.Printf("  Border flips:       %d hexes flipped\n", flipped)

	hp := make(map[int64]int, len(faction))
	for h3 := range seedableCells {
		if faction[h3] == 0 {
			continue
		}
		c, ok := candMap[h3]
		if !ok {
			continue
		}
		claim := claimNoise.Eval2(c.lng*claimFreq, c.lat*claimFreq)
		normalized := (claim - claimThreshold) / (1.0 - claimThreshold)
		hp[h3] = max(1, min(10, int(normalized*9)+1))
	}

	neutralFinal := 0
	for _, f := range faction {
		if f == 0 {
			neutralFinal++
		}
	}
	fmt.Printf("  Final:              %d neon, %d umbra, %d neutral (%s total)\n",
		countFaction(faction, 1), countFaction(faction, 2), neutralFinal, time.Since(start).Round(time.Millisecond))

	return &territoryField{faction: faction, hp: hp}
}

func buildNeighborGraph(seedable map[int64]bool) map[int64][]int64 {
	graph := make(map[int64][]int64, len(seedable))
	for h3Idx := range seedable {
		cell := h3.Cell(h3Idx)
		disk, _ := h3.GridDisk(cell, 1)
		nbrs := make([]int64, 0, 6)
		for _, n := range disk {
			ni := int64(n)
			if ni == h3Idx {
				continue
			}
			if seedable[ni] {
				nbrs = append(nbrs, ni)
			}
		}
		graph[h3Idx] = nbrs
	}
	return graph
}

func countFaction(faction map[int64]int8, f int8) int {
	c := 0
	for _, v := range faction {
		if v == f {
			c++
		}
	}
	return c
}

func seedFromCache(ctx context.Context, dbURL string, candidates []resolve, parentFeatures map[int64]features, excludedCells map[int64]bool, seedableCells map[int64]bool, tf *territoryField) seedStats {
	var stats seedStats
	var mu sync.Mutex

	workers := 8
	batchSize := 5000
	total := len(candidates)
	totalBatches := (total + batchSize - 1) / batchSize

	var wg sync.WaitGroup
	var done atomic.Int64
	seedStart := time.Now()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			wp, err := pgxpool.New(ctx, dbURL)
			if err != nil {
				return
			}
			defer wp.Close()

			for b := workerID; b < totalBatches; b += workers {
				offset := b * batchSize
				limit := min(batchSize, total-offset)
				batchCandidates := candidates[offset : offset+limit]

				batch := pgx.Batch{}
				size := 0
				localStats := seedStats{}

				flush := func() error {
					if size == 0 {
						return nil
					}
					br := wp.SendBatch(ctx, &batch)
					defer br.Close()
					for range size {
						if _, e := br.Exec(); e != nil {
							return e
						}
					}
					size = 0
					return nil
				}

				for _, c := range batchCandidates {
				if excludedCells[c.h3] {
					localStats.excluded++
					continue
				}

				if !seedableCells[c.h3] {
					localStats.nonUrban++
					continue
				}

				f, ok := parentFeatures[c.parent]
				if !ok {
					continue
				}

				factionID := tf.faction[c.h3]
				if factionID == 0 {
					localStats.neutral++
					continue
				}

				faction := "neon"
				if factionID == 2 {
					faction = "umbra"
				}
				hp := tf.hp[c.h3]
				if hp == 0 {
					hp = 1
				}
				isNatural := f.onForest || f.onMountain || f.onProtected

					batch.Queue(
						`INSERT INTO hexes (h3_index, owned_by, hp, captured_at, last_decayed_at, is_natural, lat, lng)
						 VALUES ($1,$2,$3,NOW(),NOW(),$4,$5,$6) ON CONFLICT (h3_index) DO NOTHING`,
						c.h3, faction, hp, isNatural, c.lat, c.lng,
					)
					size++
					localStats.seeded++
					if f.onForest {
						localStats.forest++
					}
					if f.onMountain {
						localStats.mountain++
					}
					if f.onProtected {
						localStats.protected++
					}
				}

				if err := flush(); err != nil {
					continue
				}

			mu.Lock()
			stats.seeded += localStats.seeded
			stats.excluded += localStats.excluded
			stats.nonUrban += localStats.nonUrban
			stats.neutral += localStats.neutral
			stats.forest += localStats.forest
			stats.mountain += localStats.mountain
			stats.protected += localStats.protected
			mu.Unlock()

			d := done.Add(int64(limit))
			if d%50000 == 0 || d >= int64(total) {
				elapsed := time.Since(seedStart)
				mu.Lock()
				s := stats
			mu.Unlock()
			fmt.Printf("  [%s] %d/%d (%.0f/s) | S:%d X:%d NU:%d Neut:%d\n",
				elapsed.Round(time.Millisecond), d, total, float64(d)/elapsed.Seconds(),
				s.seeded, s.excluded, s.nonUrban, s.neutral)
		}
		}
	}(w)
	}
	wg.Wait()
	return stats
}

const classifyQuery = `
SELECT
	EXISTS(SELECT 1 FROM osm_forest    WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1,$2), 4326))),
	EXISTS(SELECT 1 FROM osm_mountain  WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1,$2), 4326))),
	EXISTS(SELECT 1 FROM osm_protected WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1,$2), 4326)))
`

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
