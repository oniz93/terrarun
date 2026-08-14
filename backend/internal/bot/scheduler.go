package bot

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

type Scheduler struct {
	svc      *Service
	interval time.Duration
}

func NewScheduler(svc *Service, interval time.Duration) *Scheduler {
	return &Scheduler{
		svc:      svc,
		interval: interval,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Info().Dur("interval", s.interval).Msg("bot scheduler started")

	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx)
		case <-ctx.Done():
			log.Info().Msg("bot scheduler stopped")
			return
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	if !s.svc.ShouldSpawnBots(ctx) {
		return
	}

	if err := s.svc.MaintainBotBalance(ctx); err != nil {
		log.Error().Err(err).Msg("bot balance maintenance failed")
	}

	s.svc.ScheduleRuns(ctx)
}
