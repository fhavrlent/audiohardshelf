// Package scheduler decides when syncs run: on startup, on a fixed interval,
// and shortly after Audiobookshelf reports listening progress.
package scheduler

import (
	"context"
	"log/slog"
	"time"
)

type Scheduler struct {
	sync func(context.Context) error
	log  *slog.Logger

	// Interval between syncs when nothing else triggers one.
	Interval time.Duration
	// After a progress change, sync once no further changes arrive for QuietPeriod...
	QuietPeriod time.Duration
	// ...but no later than MaxDelay after the first change, so long listening
	// sessions still sync regularly.
	MaxDelay time.Duration

	changes chan struct{}
}

func New(sync func(context.Context) error, log *slog.Logger, interval time.Duration) *Scheduler {
	return &Scheduler{
		sync:        sync,
		log:         log,
		Interval:    interval,
		QuietPeriod: time.Minute,
		MaxDelay:    10 * time.Minute,
		changes:     make(chan struct{}, 1),
	}
}

// ProgressChanged records that listening progress changed. It never blocks.
func (s *Scheduler) ProgressChanged() {
	select {
	case s.changes <- struct{}{}:
	default:
	}
}

// Run syncs immediately, then whenever a sync is due, until ctx is done.
// Syncs never overlap; changes reported during a sync schedule another one.
func (s *Scheduler) Run(ctx context.Context) {
	s.runSync(ctx, "startup")

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	// Non-nil while a progress sync is pending.
	var quiet, deadline <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.changes:
			if deadline == nil {
				deadline = time.After(s.MaxDelay)
			}
			quiet = time.After(s.QuietPeriod)
			continue
		case <-quiet:
			s.runSync(ctx, "progress")
		case <-deadline:
			s.runSync(ctx, "progress")
		case <-ticker.C:
			s.runSync(ctx, "interval")
		}
		// Any sync covers all pending changes.
		quiet, deadline = nil, nil
	}
}

func (s *Scheduler) runSync(ctx context.Context, trigger string) {
	s.log.Info("sync starting", "trigger", trigger)
	if err := s.sync(ctx); err != nil {
		if ctx.Err() == nil {
			s.log.Error("sync failed", "trigger", trigger, "error", err)
		}
		return
	}
	s.log.Info("sync completed", "trigger", trigger)
}
