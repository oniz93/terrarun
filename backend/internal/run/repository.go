package run

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/terrarun/backend/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, run *domain.Run) error
	Update(ctx context.Context, run *domain.Run) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]domain.Run, int, error)
	InsertGPSPoints(ctx context.Context, points []domain.GPSPoint) error
	GetGPSPoints(ctx context.Context, runID uuid.UUID) ([]domain.GPSPoint, error)
	CountCompletedRunsInRegion(ctx context.Context, lat, lng float64, radiusKM float64, since time.Time) (int, error)
	CountCompletedRunsByUserInRegion(ctx context.Context, userID uuid.UUID, lat, lng float64, radiusKM float64, since time.Time) (int, error)
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, run *domain.Run) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO runs (id, user_id, start_time, status, social_run, social_leader, social_participants)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		run.ID, run.UserID, run.StartTime, run.Status, run.SocialRun, run.SocialLeader, run.SocialParticipants)
	return err
}

func (r *PostgresRepo) Update(ctx context.Context, run *domain.Run) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE runs SET end_time=$2, status=$3, distance_m=$4, elevation_gain_m=$5,
			avg_pace_s_per_km=$6, max_speed_kmh=$7, avg_hr_bpm=$8, max_hr_bpm=$9,
			calories=$10, closed_loop=$11, capture_mode=$12, loop_snap_distance_m=$13,
			path_buffer_radius_m=$14, polyline=$15, territory_points=$16, runner_points=$17,
			health_source=$18 WHERE id=$1`,
		run.ID, run.EndTime, run.Status, run.DistanceM, run.ElevationGainM,
		run.AvgPaceSPerKM, run.MaxSpeedKMH, run.AvgHRBPM, run.MaxHRBPM,
		run.Calories, run.ClosedLoop, run.CaptureMode, run.LoopSnapDistanceM,
		run.PathBufferRadiusM, run.Polyline, run.TerritoryPoints, run.RunnerPoints,
		run.HealthSource)
	return err
}

func scanRun(row pgx.Row) (*domain.Run, error) {
	var r domain.Run
	err := row.Scan(
		&r.ID, &r.UserID, &r.StartTime, &r.EndTime, &r.Status,
		&r.DistanceM, &r.ElevationGainM, &r.AvgPaceSPerKM, &r.MaxSpeedKMH,
		&r.AvgHRBPM, &r.MaxHRBPM, &r.Calories, &r.ClosedLoop,
		&r.CaptureMode, &r.LoopSnapDistanceM, &r.PathBufferRadiusM,
		&r.Polyline, &r.TerritoryPoints, &r.RunnerPoints, &r.SocialRun,
		&r.SocialLeader, &r.SocialParticipants, &r.HealthSource, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

const runCols = `id, user_id, start_time, end_time, status, distance_m, elevation_gain_m,
	avg_pace_s_per_km, max_speed_kmh, avg_hr_bpm, max_hr_bpm, calories, closed_loop,
	capture_mode, loop_snap_distance_m, path_buffer_radius_m, polyline, territory_points,
	runner_points, social_run, social_leader, social_participants, health_source, created_at`

func (r *PostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	row := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM runs WHERE id = $1`, runCols), id)
	return scanRun(row)
}

func (r *PostgresRepo) GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]domain.Run, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM runs WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM runs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, runCols),
		userID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []domain.Run
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, 0, err
		}
		runs = append(runs, *run)
	}
	return runs, total, nil
}

func (r *PostgresRepo) InsertGPSPoints(ctx context.Context, points []domain.GPSPoint) error {
	if len(points) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, p := range points {
		batch.Queue(
			`INSERT INTO gps_points (run_id, timestamp, lat, lng, altitude, speed, horizontal_accuracy, heart_rate)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			p.RunID, p.Timestamp, p.Lat, p.Lng, p.Altitude, p.Speed, p.HorizontalAccuracy, p.HeartRate,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range points {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepo) GetGPSPoints(ctx context.Context, runID uuid.UUID) ([]domain.GPSPoint, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT run_id, timestamp, lat, lng, altitude, speed, horizontal_accuracy, heart_rate
		 FROM gps_points WHERE run_id = $1 ORDER BY timestamp`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []domain.GPSPoint
	for rows.Next() {
		var p domain.GPSPoint
		if err := rows.Scan(&p.RunID, &p.Timestamp, &p.Lat, &p.Lng, &p.Altitude,
			&p.Speed, &p.HorizontalAccuracy, &p.HeartRate); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}

func (r *PostgresRepo) CountCompletedRunsInRegion(ctx context.Context, lat, lng float64, radiusKM float64, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM runs WHERE status = 'completed' AND created_at >= $1`, since).Scan(&count)
	return count, err
}

func (r *PostgresRepo) CountCompletedRunsByUserInRegion(ctx context.Context, userID uuid.UUID, lat, lng float64, radiusKM float64, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM runs WHERE user_id = $1 AND status = 'completed' AND created_at >= $2`,
		userID, since).Scan(&count)
	return count, err
}
