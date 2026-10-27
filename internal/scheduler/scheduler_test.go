package scheduler

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestScheduler(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		var mu sync.Mutex
		var syncs []time.Duration
		elapsed := func() []time.Duration {
			mu.Lock()
			defer mu.Unlock()
			return slices.Clone(syncs)
		}

		s := New(func(context.Context) error {
			mu.Lock()
			defer mu.Unlock()
			syncs = append(syncs, time.Since(start))
			return nil
		}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Hour)
		s.QuietPeriod = time.Minute
		s.MaxDelay = 10 * time.Minute

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go s.Run(ctx)

		// A burst of changes syncs once, a quiet period after the last change (40s + 1m).
		for range 3 {
			s.ProgressChanged()
			time.Sleep(20 * time.Second)
		}
		time.Sleep(2 * time.Minute) // until 3m

		// Changes every 45s never leave a quiet minute, so the sync happens at
		// MaxDelay (3m + 10m); the last change at 14m15s syncs at 15m15s.
		for range 16 {
			s.ProgressChanged()
			time.Sleep(45 * time.Second)
		}

		// With no changes, the interval sync runs at 1h.
		time.Sleep(time.Hour)
		synctest.Wait()

		want := []time.Duration{
			0,
			100 * time.Second,
			13 * time.Minute,
			15*time.Minute + 15*time.Second,
			time.Hour,
		}
		if got := elapsed(); !slices.Equal(got, want) {
			t.Errorf("synced at %v, want %v", got, want)
		}
	})
}
