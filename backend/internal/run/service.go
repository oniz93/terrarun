package run

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/terrarun/backend/internal/config"
	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/internal/points"
	"github.com/terrarun/backend/internal/territory"
	"github.com/terrarun/backend/internal/user"
	"github.com/terrarun/backend/internal/websocket"
)

var (
	ErrNotOwner     = errors.New("not your run")
	ErrRunNotActive = errors.New("run not active")
)

type Service struct {
	repo         Repository
	territorySvc *territory.Service
	cheatDet     *CheatDetector
	statsCalc    *StatsCalculator
	ptsCalc      *points.Calculator
	wsHub        *websocket.Hub
	userRepo     user.Repository
	cfg          *config.Config
}

func NewService(repo Repository, territorySvc *territory.Service, cheatDet *CheatDetector,
	statsCalc *StatsCalculator, ptsCalc *points.Calculator, wsHub *websocket.Hub,
	userRepo user.Repository, cfg *config.Config) *Service {
	return &Service{
		repo:         repo,
		territorySvc: territorySvc,
		cheatDet:     cheatDet,
		statsCalc:    statsCalc,
		ptsCalc:      ptsCalc,
		wsHub:        wsHub,
		userRepo:     userRepo,
		cfg:          cfg,
	}
}

func (s *Service) StartRun(ctx context.Context, userID uuid.UUID, req StartRunRequest) (*domain.Run, error) {
	run := domain.Run{
		ID:        uuid.New(),
		UserID:    userID,
		StartTime: time.Now(),
		Status:    domain.RunStatusActive,
		SocialRun: req.SocialRun,
	}

	if req.SocialRun {
		if err := s.validateSocialParticipants(ctx, userID, req); err != nil {
			return nil, err
		}
		run.SocialLeader = &userID
		run.SocialParticipants = req.SocialParticipants
	}

	if err := s.repo.Create(ctx, &run); err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	log.Info().Str("run_id", run.ID.String()).Str("user_id", userID.String()).Msg("run started")
	return &run, nil
}

func (s *Service) EndRun(ctx context.Context, userID uuid.UUID, runID uuid.UUID, req EndRunRequest) (*RunSummary, error) {
	run, err := s.repo.GetByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}
	if run.UserID != userID {
		return nil, ErrNotOwner
	}
	if run.Status != domain.RunStatusActive {
		return nil, ErrRunNotActive
	}

	cheatResult := s.cheatDet.Check(req.GPSPoints, req.DistanceM, float64(req.DurationS))
	stats := s.statsCalc.Compute(req.GPSPoints)

	var territoryPoints, hexesCaptured, hexesStolen int
	var captureMode *domain.CaptureMode

	if cheatResult == CheatClean {
		for i := range req.GPSPoints {
			req.GPSPoints[i].RunID = runID
		}

		captureResult, err := s.territorySvc.CaptureRun(ctx, run, req.GPSPoints)
		if err != nil {
			log.Warn().Err(err).Str("run_id", runID.String()).Msg("territory capture failed")
		} else {
			territoryPoints = captureResult.TerritoryPoints
			hexesCaptured = captureResult.HexesCaptured
			hexesStolen = captureResult.HexesStolen
			captureMode = &captureResult.Mode
		}
	}

	runnerPoints := s.ptsCalc.RunnerPoints(stats.DistanceM, stats.ElevationGain, territoryPoints, run.SocialRun, len(run.SocialParticipants))

	endTime := time.Now()
	run.EndTime = &endTime
	run.Status = cheatResultToRunStatus(cheatResult)
	run.DistanceM = &stats.DistanceM
	run.ElevationGainM = &stats.ElevationGain
	run.AvgPaceSPerKM = &stats.AvgPace
	run.MaxSpeedKMH = &stats.MaxSpeed
	run.TerritoryPoints = territoryPoints
	run.RunnerPoints = runnerPoints
	run.CaptureMode = captureMode
	run.Polyline = &stats.Polyline
	if captureMode != nil {
		cl := *captureMode == domain.CaptureModePolygon
		run.ClosedLoop = &cl
	}
	run.AvgHRBPM = intPtr(int(stats.AvgHR))
	run.MaxHRBPM = intPtr(int(stats.MaxHR))
	run.Calories = &stats.Calories

	if err := s.repo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("update run: %w", err)
	}

	for i := range req.GPSPoints {
		if req.GPSPoints[i].RunID == uuid.Nil {
			req.GPSPoints[i].RunID = runID
		}
	}
	if err := s.repo.InsertGPSPoints(ctx, req.GPSPoints); err != nil {
		log.Warn().Err(err).Str("run_id", runID.String()).Msg("insert gps points failed")
	}

	xpGain := int64(float64(runnerPoints) * s.cfg.XPConversionRatio)
	if xpGain > 0 {
		if err := s.userRepo.IncrementXP(ctx, userID, xpGain); err != nil {
			log.Warn().Err(err).Msg("increment xp failed")
		}
	}

	log.Info().
		Str("run_id", runID.String()).
		Str("user_id", userID.String()).
		Float64("distance", stats.DistanceM).
		Int("territory_points", territoryPoints).
		Int("runner_points", runnerPoints).
		Str("cheat", string(cheatResult)).
		Msg("run ended")

	return &RunSummary{
		RunID:           runID,
		Status:          run.Status,
		CaptureMode:     captureMode,
		DistanceM:       stats.DistanceM,
		DurationS:       int(endTime.Sub(run.StartTime).Seconds()),
		AvgPaceSPerKM:   stats.AvgPace,
		ElevationGainM:  stats.ElevationGain,
		TerritoryPoints: territoryPoints,
		RunnerPoints:    runnerPoints,
		HexesCaptured:   hexesCaptured,
		HexesStolen:     hexesStolen,
		CheatStatus:     string(cheatResult),
	}, nil
}

func (s *Service) GetRun(ctx context.Context, userID, runID uuid.UUID) (*domain.Run, error) {
	run, err := s.repo.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.UserID != userID {
		return nil, ErrNotOwner
	}
	return run, nil
}

func (s *Service) ListRuns(ctx context.Context, userID uuid.UUID, page, perPage int) ([]domain.Run, int, error) {
	return s.repo.GetByUser(ctx, userID, page, perPage)
}

func (s *Service) GetGPSPoints(ctx context.Context, userID, runID uuid.UUID) ([]domain.GPSPoint, error) {
	run, err := s.repo.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.UserID != userID {
		return nil, ErrNotOwner
	}
	return s.repo.GetGPSPoints(ctx, runID)
}

func (s *Service) validateSocialParticipants(ctx context.Context, leaderID uuid.UUID, req StartRunRequest) error {
	if len(req.SocialParticipants) == 0 {
		return fmt.Errorf("social run requires at least one participant")
	}
	if len(req.SocialParticipants) > 10 {
		return fmt.Errorf("max 10 social participants")
	}
	for _, pid := range req.SocialParticipants {
		if pid == leaderID {
			return fmt.Errorf("cannot add yourself as participant")
		}
	}
	return nil
}

type StartRunRequest struct {
	Lat                float64     `json:"lat"`
	Lng                float64     `json:"lng"`
	SocialRun          bool        `json:"social_run"`
	SocialParticipants []uuid.UUID `json:"social_participants"`
}

type EndRunRequest struct {
	GPSPoints    []domain.GPSPoint `json:"gps_points"`
	DistanceM    float64           `json:"distance_m"`
	DurationS    int               `json:"duration_s"`
	HealthSource *string           `json:"health_source"`
}

type RunSummary struct {
	RunID           uuid.UUID           `json:"run_id"`
	Status          domain.RunStatus    `json:"status"`
	CaptureMode     *domain.CaptureMode `json:"capture_mode"`
	DistanceM       float64             `json:"distance_m"`
	DurationS       int                 `json:"duration_s"`
	AvgPaceSPerKM   float64             `json:"avg_pace_s_per_km"`
	ElevationGainM  float64             `json:"elevation_gain_m"`
	TerritoryPoints int                 `json:"territory_points"`
	RunnerPoints    int                 `json:"runner_points"`
	HexesCaptured   int                 `json:"hexes_captured"`
	HexesStolen     int                 `json:"hexes_stolen"`
	CheatStatus     string              `json:"cheat_status"`
}

func intPtr(i int) *int {
	return &i
}
