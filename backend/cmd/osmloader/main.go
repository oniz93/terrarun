package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	shp "github.com/jonas-p/go-shp"
)

const (
	geofabrikBCURL  = "https://download.geofabrik.de/north-america/canada/british-columbia-latest-free.shp.zip"
	naturalEarthURL = "https://naciscdn.org/naturalearth/10m/physical/ne_10m_land.zip"
	dataDir         = "osm_data"
)

type Loader struct {
	pool *pgxpool.Pool
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://terrarun:terrarun_dev@localhost:5432/terrarun?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fatal("database connection failed: %v", err)
	}
	defer pool.Close()

	region := "british-columbia"
	if len(os.Args) > 1 {
		region = os.Args[1]
	}

	loader := &Loader{pool: pool}

	if region == "landmask" {
		if err = loader.loadGlobalLandmask(ctx); err != nil {
			fatal("landmask load failed: %v", err)
		}
		return
	}

	if region == "ocean" {
		if err = loader.loadOcean(ctx); err != nil {
			fatal("ocean load failed: %v", err)
		}
		return
	}

	if err = loader.loadRegion(ctx, region); err != nil {
		fatal("load failed: %v", err)
	}
}

func (l *Loader) loadRegion(ctx context.Context, region string) error {
	url := formatRegionURL(region)

	fmt.Printf("Downloading %s shapefiles from Geofabrik...\n", region)
	zipPath := filepath.Join(dataDir, region+".zip")
	if err := l.downloadFile(url, zipPath); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	extractDir := filepath.Join(dataDir, region)
	if err := l.extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	l.truncateTables(ctx)

	if err := l.loadWaterLayer(ctx, extractDir); err != nil {
		return fmt.Errorf("water layer: %w", err)
	}
	if err := l.loadWaterwaysLayer(ctx, extractDir); err != nil {
		return fmt.Errorf("waterways layer: %w", err)
	}
	if err := l.loadForestLayer(ctx, extractDir); err != nil {
		return fmt.Errorf("forest layer: %w", err)
	}
	if err := l.loadUrbanLayer(ctx, extractDir); err != nil {
		return fmt.Errorf("urban layer: %w", err)
	}
	if err := l.loadNaturalLayer(ctx, extractDir); err != nil {
		return fmt.Errorf("natural layer: %w", err)
	}

	fmt.Println("Fixing invalid geometries...")
	l.fixGeometries(ctx)

	fmt.Println("Creating spatial indexes...")
	l.createIndexes(ctx)

	fmt.Println("Done.")
	return nil
}

func (l *Loader) truncateTables(ctx context.Context) {
	tables := []string{"osm_water", "osm_waterways", "osm_forest", "osm_urban", "osm_protected", "osm_mountain"}
	for _, table := range tables {
		fmt.Printf("Truncating %s...\n", table)
		l.pool.Exec(ctx, "TRUNCATE TABLE "+table)
	}
}

func (l *Loader) fixGeometries(ctx context.Context) {
	tables := []string{"osm_water", "osm_waterways", "osm_forest", "osm_urban", "osm_protected", "osm_mountain"}
	for _, table := range tables {
		fmt.Printf("  fixing invalid geometries in %s...\n", table)
		sql := fmt.Sprintf("UPDATE %s SET geom = ST_MakeValid(geom) WHERE NOT ST_IsValid(geom)", table)
		result, err := l.pool.Exec(ctx, sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: fix geometries %s: %v\n", table, err)
		} else {
			fmt.Printf("  %s: fixed %d invalid geometries\n", table, result.RowsAffected())
		}
	}
}

func (l *Loader) loadWaterLayer(ctx context.Context, dir string) error {
	shpFiles := l.findShapefiles(dir, "water_a")
	if len(shpFiles) == 0 {
		return fmt.Errorf("no water_a shapefiles in %s", dir)
	}

	fmt.Println("Loading water features...")
	total := 0
	for _, sf := range shpFiles {
		n, err := l.streamLoad(ctx, sf, "osm_water", func(fclass string) (string, bool) {
			switch fclass {
			case "water", "reservoir", "river", "canal", "wetland", "glacier", "bay", "dock":
				return fclass, true
			}
			return "", false
		})
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(sf), err)
		}
		fmt.Printf("  %s: %d water features\n", filepath.Base(sf), n)
		total += n
	}
	fmt.Printf("Total water features: %d\n", total)
	return nil
}

func (l *Loader) loadWaterwaysLayer(ctx context.Context, dir string) error {
	shpFiles := l.findShapefiles(dir, "waterways_")
	if len(shpFiles) == 0 {
		fmt.Println("  no waterways_ shapefiles found, skipping")
		return nil
	}

	fmt.Println("Loading waterway lines (rivers, streams, canals)...")
	total := 0
	for _, sf := range shpFiles {
		n, err := l.streamLoad(ctx, sf, "osm_waterways", func(fclass string) (string, bool) {
			switch fclass {
			case "river", "canal", "stream", "drain", "ditch":
				return fclass, true
			}
			return "", false
		})
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(sf), err)
		}
		fmt.Printf("  %s: %d waterway lines\n", filepath.Base(sf), n)
		total += n
	}
	fmt.Printf("Total waterway lines: %d\n", total)
	return nil
}

func (l *Loader) loadForestLayer(ctx context.Context, dir string) error {
	shpFiles := l.findShapefiles(dir, "landuse_a")
	if len(shpFiles) > 0 {
		fmt.Println("Loading forest features...")
		total := 0
		for _, sf := range shpFiles {
			n, err := l.streamLoad(ctx, sf, "osm_forest", func(fclass string) (string, bool) {
				if fclass == "forest" {
					return "forest", true
				}
				return "", false
			})
			if err != nil {
				return fmt.Errorf("%s: %w", filepath.Base(sf), err)
			}
			fmt.Printf("  %s: %d forest features\n", filepath.Base(sf), n)
			total += n
		}
		fmt.Printf("Total forest features: %d\n", total)
	}

	shpNatural := l.findShapefiles(dir, "natural_a")
	for _, sf := range shpNatural {
		n, err := l.streamLoad(ctx, sf, "osm_forest", func(fclass string) (string, bool) {
			switch fclass {
			case "wood", "scrub", "heath", "grassland", "meadow":
				return fclass, true
			}
			return "", false
		})
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(sf), err)
		}
		fmt.Printf("  %s: %d natural woodland features\n", filepath.Base(sf), n)
	}
	return nil
}

func (l *Loader) loadUrbanLayer(ctx context.Context, dir string) error {
	shpFiles := l.findShapefiles(dir, "landuse_a")
	if len(shpFiles) == 0 {
		fmt.Println("  no landuse_a shapefiles found, skipping urban")
		return nil
	}

	fmt.Println("Loading urban landuse polygons...")
	total := 0
	for _, sf := range shpFiles {
		n, err := l.streamLoad(ctx, sf, "osm_urban", func(fclass string) (string, bool) {
			switch fclass {
			case "residential", "commercial", "industrial", "retail",
				"cemetery", "recreation_ground", "allotments", "military":
				return fclass, true
			}
			return "", false
		})
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(sf), err)
		}
		fmt.Printf("  %s: %d urban features\n", filepath.Base(sf), n)
		total += n
	}
	fmt.Printf("Total urban features: %d\n", total)
	return nil
}

func (l *Loader) loadNaturalLayer(ctx context.Context, dir string) error {
	shpFiles := l.findShapefiles(dir, "natural_a")
	if len(shpFiles) == 0 {
		return nil
	}

	for _, sf := range shpFiles {
		wn, err := l.streamLoad(ctx, sf, "osm_water", func(fclass string) (string, bool) {
			switch fclass {
			case "water", "wetland", "glacier", "bay":
				return fclass, true
			}
			return "", false
		})
		if err != nil {
			return fmt.Errorf("%s water: %w", filepath.Base(sf), err)
		}

		pn, err := l.streamLoad(ctx, sf, "osm_protected", func(fclass string) (string, bool) {
			switch fclass {
			case "national_park", "nature_reserve":
				return fclass, true
			}
			return "", false
		})
		if err != nil {
			return fmt.Errorf("%s protected: %w", filepath.Base(sf), err)
		}

		mn, err := l.streamLoad(ctx, sf, "osm_mountain", func(fclass string) (string, bool) {
			switch fclass {
			case "peak", "bare_rock", "scree", "ridge":
				return fclass, true
			}
			return "", false
		})
		if err != nil {
			return fmt.Errorf("%s mountain: %w", filepath.Base(sf), err)
		}

		fmt.Printf("  %s: %d water, %d protected, %d mountain\n",
			filepath.Base(sf), wn, pn, mn)
	}
	return nil
}

func (l *Loader) loadGlobalLandmask(ctx context.Context) error {
	fmt.Println("Downloading Natural Earth 10m land polygons...")
	zipPath := filepath.Join(dataDir, "ne_10m_land.zip")
	if err := l.downloadFile(naturalEarthURL, zipPath); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	extractDir := filepath.Join(dataDir, "natural_earth")
	if err := l.extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	shpFiles := l.findShapefiles(extractDir, "land")
	if len(shpFiles) == 0 {
		return fmt.Errorf("no land shapefile in %s", extractDir)
	}

	fmt.Println("Loading global landmask polygons...")
	total := 0
	for _, sf := range shpFiles {
		n, err := l.streamLoadNoClass(ctx, sf, "global_landmask")
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(sf), err)
		}
		total += n
	}

	fmt.Printf("Total land polygons: %d\n", total)

	fmt.Println("Creating GIST index...")
	l.pool.Exec(ctx, "CREATE INDEX IF NOT EXISTS idx_global_landmask_geom ON global_landmask USING GIST(geom)")
	fmt.Println("Landmask complete.")
	return nil
}

func (l *Loader) loadOcean(ctx context.Context) error {
	oceanDir := filepath.Join(dataDir, "ocean")
	zipPath := filepath.Join(oceanDir, "water-polygons-split-4326.zip")

	if _, err := os.Stat(zipPath); err != nil {
		return fmt.Errorf("ocean zip not found at %s — download from https://osmdata.openstreetmap.de/download/water-polygons-split-4326.zip", zipPath)
	}

	extractDir := filepath.Join(oceanDir, "extracted")
	if err := l.extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	shpFiles := l.findShapefilesRecursive(extractDir, "water_polygons")
	if len(shpFiles) == 0 {
		return fmt.Errorf("no water_polygons shapefile in %s", extractDir)
	}

	fmt.Println("Truncating osm_ocean...")
	l.pool.Exec(ctx, "TRUNCATE TABLE osm_ocean")

	bcMinLng, bcMinLat, bcMaxLng, bcMaxLat := -139.0, 48.0, -114.0, 60.0
	fmt.Printf("Filtering to BC bbox [%.1f, %.1f] to [%.1f, %.1f]\n", bcMinLng, bcMinLat, bcMaxLng, bcMaxLat)

	total := 0
	for _, sf := range shpFiles {
		n, err := l.streamLoadOcean(ctx, sf, "osm_ocean", bcMinLng, bcMinLat, bcMaxLng, bcMaxLat)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(sf), err)
		}
		fmt.Printf("  %s: %d ocean polygons (in BC bbox)\n", filepath.Base(sf), n)
		total += n
	}

	fmt.Printf("Total ocean polygons: %d\n", total)

	fmt.Println("Creating GIST index...")
	l.pool.Exec(ctx, "CREATE INDEX IF NOT EXISTS idx_osm_ocean_geom ON osm_ocean USING GIST(geom)")
	fmt.Println("Ocean load complete.")
	return nil
}

func (l *Loader) streamLoad(ctx context.Context, shpPath, table string, filter func(fclass string) (string, bool)) (int, error) {
	reader, err := shp.Open(shpPath)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	fclassIdx := -1
	for i, field := range reader.Fields() {
		if strings.EqualFold(field.String(), "fclass") {
			fclassIdx = i
			break
		}
	}

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	tmpTable := "_osm_tmp_wkt1"
	if _, err := tx.Exec(ctx, "DROP TABLE IF EXISTS "+tmpTable); err != nil {
		return 0, fmt.Errorf("drop temp table: %w", err)
	}
	if _, err := tx.Exec(ctx, "CREATE TEMP TABLE "+tmpTable+" (wkt TEXT, class TEXT)"); err != nil {
		return 0, fmt.Errorf("create temp table: %w", err)
	}

	src := &copySource{
		reader:    reader,
		fclassIdx: fclassIdx,
		filter:    filter,
	}

	start := time.Now()
	copied, err := tx.CopyFrom(ctx, pgx.Identifier{tmpTable}, []string{"wkt", "class"}, src)
	if err != nil {
		return src.count, fmt.Errorf("copy from: %w", err)
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Printf("    loaded %d WKT rows in %s (%.0f rows/s), converting to geometry...\n", copied, elapsed, float64(copied)/elapsed.Seconds())

	convSQL := fmt.Sprintf("INSERT INTO %s (geom, class) SELECT ST_GeomFromText(wkt, 4326), class FROM %s", table, tmpTable)
	result, err := tx.Exec(ctx, convSQL)
	if err != nil {
		return int(copied), fmt.Errorf("geometry conversion: %w", err)
	}
	fmt.Printf("    converted to geometry: %d rows\n", result.RowsAffected())

	if err := tx.Commit(ctx); err != nil {
		return int(copied), fmt.Errorf("commit: %w", err)
	}

	return int(copied), src.Err()
}

func (l *Loader) streamLoadNoClass(ctx context.Context, shpPath, table string) (int, error) {
	reader, err := shp.Open(shpPath)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	tmpTable := "_osm_tmp_wkt2"
	if _, err := tx.Exec(ctx, "DROP TABLE IF EXISTS "+tmpTable); err != nil {
		return 0, fmt.Errorf("drop temp table: %w", err)
	}
	if _, err := tx.Exec(ctx, "CREATE TEMP TABLE "+tmpTable+" (wkt TEXT)"); err != nil {
		return 0, fmt.Errorf("create temp table: %w", err)
	}

	src := &copySource{
		reader: reader,
	}

	start := time.Now()
	copied, err := tx.CopyFrom(ctx, pgx.Identifier{tmpTable}, []string{"wkt"}, src)
	if err != nil {
		return src.count, fmt.Errorf("copy from: %w", err)
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Printf("    loaded %d WKT rows in %s (%.0f rows/s), converting to geometry...\n", copied, elapsed, float64(copied)/elapsed.Seconds())

	convSQL := fmt.Sprintf("INSERT INTO %s (geom) SELECT ST_Multi(ST_GeomFromText(wkt, 4326)) FROM %s", table, tmpTable)
	result, err := tx.Exec(ctx, convSQL)
	if err != nil {
		return int(copied), fmt.Errorf("geometry conversion: %w", err)
	}
	fmt.Printf("    converted to geometry: %d rows\n", result.RowsAffected())

	if err := tx.Commit(ctx); err != nil {
		return int(copied), fmt.Errorf("commit: %w", err)
	}

	return int(copied), src.Err()
}

func (l *Loader) streamLoadOcean(ctx context.Context, shpPath, table string, minLng, minLat, maxLng, maxLat float64) (int, error) {
	reader, err := shp.Open(shpPath)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	tmpTable := "_osm_tmp_ocean"
	if _, err := tx.Exec(ctx, "DROP TABLE IF EXISTS "+tmpTable); err != nil {
		return 0, fmt.Errorf("drop temp table: %w", err)
	}
	if _, err := tx.Exec(ctx, "CREATE TEMP TABLE "+tmpTable+" (wkt TEXT)"); err != nil {
		return 0, fmt.Errorf("create temp table: %w", err)
	}

	src := &copySource{
		reader:  reader,
		bbox:    [4]float64{minLng, minLat, maxLng, maxLat},
		useBbox: true,
	}

	start := time.Now()
	copied, err := tx.CopyFrom(ctx, pgx.Identifier{tmpTable}, []string{"wkt"}, src)
	if err != nil {
		return src.count, fmt.Errorf("copy from: %w", err)
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Printf("    loaded %d WKT rows in %s (%.0f rows/s), converting to geometry...\n", copied, elapsed, float64(copied)/elapsed.Seconds())

	convSQL := fmt.Sprintf("INSERT INTO %s (geom) SELECT ST_Multi(ST_GeomFromText(wkt, 4326)) FROM %s", table, tmpTable)
	result, err := tx.Exec(ctx, convSQL)
	if err != nil {
		return int(copied), fmt.Errorf("geometry conversion: %w", err)
	}
	fmt.Printf("    converted to geometry: %d rows\n", result.RowsAffected())

	if err := tx.Commit(ctx); err != nil {
		return int(copied), fmt.Errorf("commit: %w", err)
	}

	return int(copied), src.Err()
}

type copySource struct {
	reader    *shp.Reader
	fclassIdx int
	filter    func(fclass string) (string, bool)
	curGeom   string
	curClass  string
	noClass   bool
	count     int
	bbox      [4]float64
	useBbox   bool
}

func (s *copySource) Next() bool {
	for s.reader.Next() {
		_, shape := s.reader.Shape()

		var wkt string
		switch sh := shape.(type) {
		case *shp.PolyLine:
			if s.useBbox && !bboxOverlaps(sh.Box, s.bbox) {
				continue
			}
			wkt = polyLineToWKT(sh)
		case *shp.Polygon:
			if s.useBbox && !bboxOverlaps(sh.Box, s.bbox) {
				continue
			}
			wkt = polyToWKT(sh)
		default:
			continue
		}
		if wkt == "" {
			continue
		}

		if s.filter != nil {
			fclass := ""
			if s.fclassIdx >= 0 {
				fclass = strings.TrimSpace(s.reader.Attribute(s.fclassIdx))
			}
			class, match := s.filter(fclass)
			if !match {
				continue
			}
			s.curClass = class
		}

		s.curGeom = wkt
		s.count++

		if s.count%10000 == 0 {
			fmt.Printf("    scanned %d features...\n", s.count)
		}

		return true
	}
	return false
}

func (s *copySource) Values() ([]any, error) {
	if s.filter != nil {
		return []any{s.curGeom, s.curClass}, nil
	}
	return []any{s.curGeom}, nil
}

func (s *copySource) Err() error {
	return s.reader.Err()
}

func (l *Loader) createIndexes(ctx context.Context) {
	tables := map[string]string{
		"osm_water":      "idx_osm_water_geom",
		"osm_waterways":  "idx_osm_waterways_geom",
		"osm_forest":     "idx_osm_forest_geom",
		"osm_urban":      "idx_osm_urban_geom",
		"osm_protected":  "idx_osm_protected_geom",
		"osm_mountain":   "idx_osm_mountain_geom",
	}
	for table, idx := range tables {
		fmt.Printf("  creating index %s on %s...\n", idx, table)
		sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING GIST(geom)", idx, table)
		if _, err := l.pool.Exec(ctx, sql); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: index %s: %v\n", idx, err)
		}
	}
}

func (l *Loader) downloadFile(url, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		info, _ := os.Stat(dest)
		fmt.Printf("  %s already downloaded (%.0f MB), skipping\n", filepath.Base(dest), float64(info.Size())/(1024*1024))
		return nil
	}

	os.MkdirAll(filepath.Dir(dest), 0755)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	start := time.Now()
	written, err := io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(dest)
		return err
	}
	fmt.Printf("  downloaded %.1f MB in %s\n", float64(written)/(1024*1024), time.Since(start).Round(time.Second))
	return nil
}

func (l *Loader) extractZip(src, dest string) error {
	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		files, _ := os.ReadDir(dest)
		if len(files) > 0 {
			fmt.Printf("  %s already extracted (%d files), skipping\n", dest, len(files))
			return nil
		}
	}

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(dest, 0755)

	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(target), 0755)

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}

	fmt.Printf("  extracted %d files\n", len(r.File))
	return nil
}

func (l *Loader) findShapefiles(dir, pattern string) []string {
	var results []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return results
	}
	lower := strings.ToLower(pattern)
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.Contains(name, lower) && strings.HasSuffix(name, ".shp") {
			results = append(results, filepath.Join(dir, entry.Name()))
		}
	}
	return results
}

func (l *Loader) findShapefilesRecursive(dir, pattern string) []string {
	var results []string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.Contains(name, strings.ToLower(pattern)) && strings.HasSuffix(name, ".shp") {
			results = append(results, path)
		}
		return nil
	})
	return results
}

func formatRegionURL(region string) string {
	switch region {
	case "british-columbia", "bc":
		return geofabrikBCURL
	default:
		return fmt.Sprintf("https://download.geofabrik.de/north-america/canada/%s-latest-free.shp.zip", region)
	}
}

func polyToWKT(p *shp.Polygon) string {
	if len(p.Parts) == 0 || len(p.Points) < 3 {
		return ""
	}

	if len(p.Parts) == 1 {
		return "POLYGON" + ringToWKT(p.Points)
	}

	var rings []string
	for i, start := range p.Parts {
		end := int32(len(p.Points))
		if i < len(p.Parts)-1 {
			end = p.Parts[i+1]
		}
		if end-start >= 3 {
			rings = append(rings, ringToWKT(p.Points[start:end]))
		}
	}

	if len(rings) == 0 {
		return ""
	}
	if len(rings) == 1 {
		return "POLYGON" + rings[0]
	}
	return "MULTIPOLYGON(" + strings.Join(rings, ",") + ")"
}

func polyLineToWKT(p *shp.PolyLine) string {
	if len(p.Parts) == 0 || len(p.Points) < 2 {
		return ""
	}

	if len(p.Parts) == 1 {
		return "LINESTRING" + lineToWKT(p.Points)
	}

	var lines []string
	for i, start := range p.Parts {
		end := int32(len(p.Points))
		if i < len(p.Parts)-1 {
			end = p.Parts[i+1]
		}
		if end-start >= 2 {
			lines = append(lines, lineToWKT(p.Points[start:end]))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	if len(lines) == 1 {
		return "LINESTRING" + lines[0]
	}
	return "MULTILINESTRING(" + strings.Join(lines, ",") + ")"
}

func lineToWKT(points []shp.Point) string {
	var coords []string
	for _, pt := range points {
		coords = append(coords,
			strconv.FormatFloat(pt.X, 'f', 8, 64)+" "+
				strconv.FormatFloat(pt.Y, 'f', 8, 64))
	}
	return "(" + strings.Join(coords, ",") + ")"
}

func bboxOverlaps(box shp.Box, filter [4]float64) bool {
	return box.MinX <= filter[2] && box.MaxX >= filter[0] &&
		box.MinY <= filter[3] && box.MaxY >= filter[1]
}

func ringToWKT(points []shp.Point) string {
	var coords []string
	for _, pt := range points {
		coords = append(coords,
			strconv.FormatFloat(pt.X, 'f', 8, 64)+" "+
				strconv.FormatFloat(pt.Y, 'f', 8, 64))
	}
	return "((" + strings.Join(coords, ",") + "))"
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
