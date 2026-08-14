package territory

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LandRepository interface {
	IsOnLand(ctx context.Context, lat, lng float64) (bool, error)
	IsWater(ctx context.Context, lat, lng float64) (bool, error)
	IsForest(ctx context.Context, lat, lng float64) (bool, error)
	IsProtected(ctx context.Context, lat, lng float64) (bool, error)
	IsMountain(ctx context.Context, lat, lng float64) (bool, error)
	ClassifyPoint(ctx context.Context, lat, lng float64) (*OSMPointFeatures, error)

	GetLandGeometry(ctx context.Context, parentIndex int64) ([]byte, error)
	SaveLandGeoJSON(ctx context.Context, parentIndex int64, geojson string) error
	SaveWaterGeoJSON(ctx context.Context, parentIndex int64, geojson string) error
}

type OSMPointFeatures struct {
	IsLand      bool
	IsWater     bool
	IsForest    bool
	IsMountain  bool
	IsProtected bool
}

type PostgresLandRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresLandRepo(pool *pgxpool.Pool) *PostgresLandRepo {
	return &PostgresLandRepo{pool: pool}
}

func (r *PostgresLandRepo) IsOnLand(ctx context.Context, lat, lng float64) (bool, error) {
	var onLand bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM global_landmask WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&onLand)
	return onLand, err
}

func (r *PostgresLandRepo) IsWater(ctx context.Context, lat, lng float64) (bool, error) {
	var isWater bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_water WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&isWater)
	return isWater, err
}

func (r *PostgresLandRepo) IsForest(ctx context.Context, lat, lng float64) (bool, error) {
	var isForest bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_forest WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&isForest)
	return isForest, err
}

func (r *PostgresLandRepo) IsProtected(ctx context.Context, lat, lng float64) (bool, error) {
	var isProtected bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_protected WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&isProtected)
	return isProtected, err
}

func (r *PostgresLandRepo) IsMountain(ctx context.Context, lat, lng float64) (bool, error) {
	var isMountain bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_mountain WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&isMountain)
	return isMountain, err
}

func (r *PostgresLandRepo) ClassifyPoint(ctx context.Context, lat, lng float64) (*OSMPointFeatures, error) {
	pf := &OSMPointFeatures{}

	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM global_landmask WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsLand)
	if err != nil {
		return pf, err
	}

	err = r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_water WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsWater)
	if err != nil {
		return pf, err
	}

	err = r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_forest WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsForest)
	if err != nil {
		pf.IsForest = false
	}

	err = r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_mountain WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsMountain)
	if err != nil {
		pf.IsMountain = false
	}

	err = r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_protected WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsProtected)
	if err != nil {
		pf.IsProtected = false
	}

	return pf, nil
}

func (r *PostgresLandRepo) GetLandGeometry(ctx context.Context, parentIndex int64) ([]byte, error) {
	var wkb []byte
	err := r.pool.QueryRow(ctx, "SELECT ST_AsBinary(geom) FROM land_cache WHERE h3_parent_index = $1", parentIndex).Scan(&wkb)
	if err != nil {
		return nil, err
	}
	return wkb, nil
}

func (r *PostgresLandRepo) SaveLandGeoJSON(ctx context.Context, parentIndex int64, geojson string) error {
	query := `
		INSERT INTO land_cache (h3_parent_index, geom) 
		VALUES ($1, (
			SELECT ST_Multi(ST_Union(geom))
			FROM (
				SELECT ST_GeomFromGeoJSON(feat->>'geometry') as geom
				FROM jsonb_array_elements($2::jsonb->'features') as feat
			) f
			WHERE ST_GeometryType(geom) IN ('ST_Polygon', 'ST_MultiPolygon')
		)) 
		ON CONFLICT (h3_parent_index) DO NOTHING`
	_, err := r.pool.Exec(ctx, query, parentIndex, geojson)
	return err
}

func (r *PostgresLandRepo) SaveWaterGeoJSON(ctx context.Context, parentIndex int64, geojson string) error {
	query := `
		INSERT INTO water_cache (h3_parent_index, geom) 
		VALUES ($1, (
			SELECT ST_Multi(ST_Union(geom))
			FROM (
				SELECT ST_GeomFromGeoJSON(feat->>'geometry') as geom
				FROM jsonb_array_elements($2::jsonb->'features') as feat
			) f
			WHERE ST_GeometryType(geom) IN ('ST_Polygon', 'ST_MultiPolygon')
		)) 
		ON CONFLICT (h3_parent_index) DO NOTHING`
	_, err := r.pool.Exec(ctx, query, parentIndex, geojson)
	return err
}
