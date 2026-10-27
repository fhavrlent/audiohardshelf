// Package syncer pushes Audiobookshelf listening progress to Hardcover.
package syncer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/fhavrlent/audiohardshelf/internal/audiobookshelf"
	"github.com/fhavrlent/audiohardshelf/internal/hardcover"
	"github.com/fhavrlent/audiohardshelf/internal/httpjson"
)

// Books finished within this window are synced so they get marked as read on Hardcover.
const finishedLookback = 30 * 24 * time.Hour

type Audiobookshelf interface {
	Ping(ctx context.Context) error
	CurrentlyListening(ctx context.Context, finishedSince time.Time) ([]audiobookshelf.MediaProgress, error)
	Book(ctx context.Context, libraryItemID string) (audiobookshelf.Book, error)
}

type Hardcover interface {
	Me(ctx context.Context) (hardcover.User, error)
	CurrentlyReadingAudiobooks(ctx context.Context, userID int) ([]hardcover.Audiobook, error)
	LatestRead(ctx context.Context, userBookID int) (*hardcover.Read, error)
	UpdateRead(ctx context.Context, readID int, update hardcover.ReadUpdate) error
	MarkAsRead(ctx context.Context, userBookID, editionID int) error
}

type result int

const (
	updated result = iota
	skipped
	notFound
	failed
)

type Syncer struct {
	abs Audiobookshelf
	hc  Hardcover
	log *slog.Logger
	now func() time.Time
}

func New(abs Audiobookshelf, hc Hardcover, log *slog.Logger) *Syncer {
	return &Syncer{abs: abs, hc: hc, log: log, now: time.Now}
}

// Run performs one sync. Per-book failures are logged and counted; an error is
// returned only when the sync could not run at all.
func (s *Syncer) Run(ctx context.Context) error {
	if err := s.abs.Ping(ctx); err != nil {
		return fmt.Errorf("connecting to Audiobookshelf: %w%s", err, hint(err, "ABS_URL", "ABS_API_KEY"))
	}

	listening, err := s.abs.CurrentlyListening(ctx, s.now().Add(-finishedLookback))
	if err != nil {
		return fmt.Errorf("fetching books in progress from Audiobookshelf: %w%s", err, hint(err, "ABS_URL", "ABS_API_KEY"))
	}
	s.log.Info("found books on Audiobookshelf", "inProgressOrRecentlyFinished", len(listening))
	if len(listening) == 0 {
		return nil
	}

	user, err := s.hc.Me(ctx)
	if err != nil {
		return fmt.Errorf("connecting to Hardcover: %w%s", err, hint(err, "HARDCOVER_API_URL", "HARDCOVER_API_KEY"))
	}
	s.log.Info("connected to Hardcover", "user", user.Username, "userId", user.ID)

	audiobooks, err := s.hc.CurrentlyReadingAudiobooks(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("fetching currently reading audiobooks from Hardcover: %w%s", err, hint(err, "HARDCOVER_API_URL", "HARDCOVER_API_KEY"))
	}
	s.log.Info("found audiobooks currently reading on Hardcover", "count", len(audiobooks))

	counts := map[result]int{}
	seen := map[string]bool{}
	for _, progress := range listening {
		if seen[progress.LibraryItemID] {
			continue
		}
		seen[progress.LibraryItemID] = true
		counts[s.syncBook(ctx, progress, audiobooks)]++
	}

	s.log.Info("sync finished",
		"updated", counts[updated], "skipped", counts[skipped],
		"notFound", counts[notFound], "failed", counts[failed])
	return nil
}

func (s *Syncer) syncBook(ctx context.Context, progress audiobookshelf.MediaProgress, audiobooks []hardcover.Audiobook) result {
	log := s.log.With("libraryItemId", progress.LibraryItemID)

	// Books count as in progress when only their ebook was read; their listening
	// position is 0 and must not overwrite audiobook progress on Hardcover.
	if !progress.IsFinished && progress.CurrentTime < 1 {
		log.Debug("no listening progress yet")
		return skipped
	}

	book, err := s.abs.Book(ctx, progress.LibraryItemID)
	if err != nil {
		log.Error("fetching book details failed", "error", err)
		return failed
	}
	if book.Title == "" {
		book.Title = "Unknown Title"
	}
	log = log.With("title", book.Title)

	audiobook, matchedBy := findMatch(book, audiobooks)
	if audiobook == nil {
		if progress.IsFinished {
			log.Debug("finished book is not currently reading on Hardcover")
			return skipped
		}
		log.Warn("no matching audiobook currently reading on Hardcover")
		return notFound
	}
	log = log.With("editionId", audiobook.EditionID)
	log.Debug("matched audiobook on Hardcover", "by", matchedBy)

	read, err := s.hc.LatestRead(ctx, audiobook.UserBookID)
	if err != nil {
		log.Error("fetching read from Hardcover failed", "error", err)
		return failed
	}
	if read == nil {
		log.Error("book has no read on Hardcover")
		return failed
	}

	finishedAt := formatDate(progress.FinishedAt)
	// A read started after the book was finished is a re-read; don't mark it as read.
	if progress.IsFinished && finishedAt != "" && read.StartedAt > finishedAt {
		log.Debug("Hardcover read started after the book was finished, treating it as a re-read",
			"readStartedAt", read.StartedAt, "finishedAt", finishedAt)
		return skipped
	}

	seconds := int(math.Round(progress.CurrentTime))
	if read.ProgressSeconds == seconds && !progress.IsFinished {
		log.Info("progress unchanged", "seconds", seconds)
		return skipped
	}

	err = s.hc.UpdateRead(ctx, read.ID, hardcover.ReadUpdate{
		EditionID:       audiobook.EditionID,
		ProgressSeconds: seconds,
		StartedAt:       formatDate(progress.StartedAt),
		FinishedAt:      finishedAt,
	})
	if err != nil {
		log.Error("updating progress on Hardcover failed", "error", err)
		return failed
	}
	log.Info("updated progress", "seconds", seconds)

	if progress.IsFinished {
		if err := s.hc.MarkAsRead(ctx, audiobook.UserBookID, audiobook.EditionID); err != nil {
			log.Warn("marking book as read on Hardcover failed", "error", err)
		} else {
			log.Info("marked book as read")
		}
	}
	return updated
}

// formatDate formats epoch milliseconds as a local YYYY-MM-DD date, or "" for zero.
func formatDate(ms int64) string {
	if ms == 0 {
		return ""
	}
	return time.UnixMilli(ms).Format(time.DateOnly)
}

// hint suggests which settings to check for common HTTP failures.
func hint(err error, urlSetting, keySetting string) string {
	var statusErr *httpjson.StatusError
	if !errors.As(err, &statusErr) {
		return ""
	}
	switch statusErr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Sprintf(" (check that %s is valid)", keySetting)
	case http.StatusNotFound:
		return fmt.Sprintf(" (check %s)", urlSetting)
	}
	return ""
}
