package territory

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// DecayScheduler periodically applies the daily HP decay to owned hexes.
// BatchDecayHP only touches hexes that haven't decayed yet today, so the
// scheduler can safely run more frequently than daily.
type DecayScheduler struct {
	svc       *Service
	batchSize int
	interval  time.Duration
}

func NewDecayScheduler(svc *Service, batchSize int, interval time.Duration) *DecayScheduler {
	return &DecayScheduler{
		svc:       svc,
		batchSize: batchSize,
		interval:  interval,
	}
}

func (s *DecayScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Info().Dur("interval", s.interval).Msg("hp decay scheduler started")

	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx)
		case <-ctx.Done():
			log.Info().Msg("hp decay scheduler stopped")
			return
		}
	}
}

func (s *DecayScheduler) runOnce(ctx context.Context) {
	affected, err := s.svc.DecayAllHexes(ctx, s.batchSize)
	if err != nil {
		log.Error().Err(err).Msg("hp decay failed")
		return
	}
	if affected > 0 {
		log.Info().Int("hexes", affected).Msg("hp decay applied")
	}
}
