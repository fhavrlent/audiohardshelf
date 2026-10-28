// Package syncer pushes Audiobookshelf listening progress to Hardcover.
package syncer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/fhavrlent/audiohardshelf/internal/audiobookshelf"
	"github.com/fhavrlent/audiohardshelf/internal/hardcover"
	"github.com/fhavrlent/audiohardshelf/internal/httpjson"
	"github.com/fhavrlent/audiohardshelf/internal/state"
)

const (
	// Books finished within this window are synced so they get marked as read on
	// Hardcover. When the last sync was longer ago, the window reaches back to it.
	finishedLookback = 30 * 24 * time.Hour
	// Only books listened to within this window are started on Hardcover, so
	// abandoned books left in progress on Audiobookshelf aren't.
	autoStartRecency = 14 * 24 * time.Hour
	// A book that couldn't be started on Hardcover isn't tried again for this long,
	// so it isn't looked up after every listen.
	autoStartRetry = 24 * time.Hour
	// How many Hardcover search results are checked when starting a book by title.
	searchResults = 10
	// Progress is scaled to the Hardcover edition's length when the lengths differ
	// by more than this fraction, so the percentage on Hardcover is right.
	durationTolerance = 0.05
)

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

	// Used to start books on Hardcover.
	FindAudioEdition(ctx context.Context, isbn, asin string) (*hardcover.Edition, error)
	SearchBooks(ctx context.Context, query string, limit int) ([]int, error)
	CatalogBooks(ctx context.Context, ids []int) ([]hardcover.CatalogBook, error)
	UserBook(ctx context.Context, userID, bookID int) (*hardcover.UserBook, error)
	AddCurrentlyReading(ctx context.Context, bookID, editionID int) (int, error)
	StartReading(ctx context.Context, userBookID, editionID int) error
	InsertRead(ctx context.Context, userBookID int, read hardcover.ReadUpdate) (int, error)
}

type result int

const (
	updated result = iota
	skipped
	notFound
	failed
)

type Options struct {
	// AutoStart puts books being listened to on Hardcover as Currently Reading
	// in their audiobook edition when they aren't already.
	AutoStart bool
	// State remembers what was synced; nil keeps it in memory.
	State *state.Store
}

type Syncer struct {
	abs       Audiobookshelf
	hc        Hardcover
	log       *slog.Logger
	now       func() time.Time
	autoStart bool
	state     *state.Store
}

func New(abs Audiobookshelf, hc Hardcover, log *slog.Logger, opts Options) *Syncer {
	store := opts.State
	if store == nil {
		store = state.Memory()
	}
	return &Syncer{abs: abs, hc: hc, log: log, now: time.Now, autoStart: opts.AutoStart, state: store}
}

// run holds what one sync fetched from Hardcover.
type run struct {
	userID     int
	audiobooks []hardcover.Audiobook
}

// Run performs one sync. Per-book failures are logged and counted; an error is
// returned only when the sync could not run at all.
func (s *Syncer) Run(ctx context.Context) error {
	started := s.now()
	if err := s.abs.Ping(ctx); err != nil {
		return fmt.Errorf("connecting to Audiobookshelf: %w%s", err, hint(err, "ABS_URL", "ABS_API_KEY"))
	}

	finishedSince := started.Add(-finishedLookback)
	if last := s.state.LastSync; !last.IsZero() && last.Before(finishedSince) {
		finishedSince = last.Add(-24 * time.Hour)
	}
	listening, err := s.abs.CurrentlyListening(ctx, finishedSince)
	if err != nil {
		return fmt.Errorf("fetching books in progress from Audiobookshelf: %w%s", err, hint(err, "ABS_URL", "ABS_API_KEY"))
	}
	s.log.Info("found books on Audiobookshelf", "inProgressOrRecentlyFinished", len(listening))
	if len(listening) == 0 {
		s.finish(started, map[string]bool{})
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

	r := &run{userID: user.ID, audiobooks: audiobooks}
	counts := map[result]int{}
	seen := map[string]bool{}
	for _, progress := range listening {
		if seen[progress.LibraryItemID] {
			continue
		}
		seen[progress.LibraryItemID] = true
		counts[s.syncBook(ctx, progress, r)]++
	}

	s.finish(started, seen)
	s.log.Info("sync finished",
		"updated", counts[updated], "skipped", counts[skipped],
		"notFound", counts[notFound], "failed", counts[failed])
	return nil
}

// finish records a completed sync, forgets books no longer in progress or
// recently finished, and saves the state.
func (s *Syncer) finish(started time.Time, seen map[string]bool) {
	s.state.LastSync = started
	for id := range s.state.Books {
		if !seen[id] {
			delete(s.state.Books, id)
		}
	}
	if err := s.state.Save(); err != nil {
		s.log.Warn("saving state failed", "error", err)
	}
}

func (s *Syncer) syncBook(ctx context.Context, progress audiobookshelf.MediaProgress, r *run) result {
	log := s.log.With("libraryItemId", progress.LibraryItemID)

	// Books count as in progress when only their ebook was read; their listening
	// position is 0 and must not overwrite audiobook progress on Hardcover.
	if !progress.IsFinished && progress.CurrentTime < 1 {
		log.Debug("no listening progress yet")
		return skipped
	}

	listened := int(math.Round(progress.CurrentTime))
	last := s.state.Books[progress.LibraryItemID]
	if progress.IsFinished && progress.FinishedAt != 0 && last.MarkedReadAt == progress.FinishedAt {
		log.Debug("already marked as read")
		return skipped
	}
	if !progress.IsFinished && last.UserBookID != 0 && last.Seconds == listened && r.currentlyReading(last.UserBookID) {
		log.Debug("progress unchanged since last sync", "seconds", listened)
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

	audiobook, matchedBy := findMatch(book, r.audiobooks)
	if audiobook == nil && progress.IsFinished {
		log.Debug("finished book is not currently reading on Hardcover")
		return skipped
	}
	if audiobook == nil && s.autoStart {
		switch {
		case progress.LastUpdate < s.now().Add(-autoStartRecency).UnixMilli():
			log.Debug("not starting book on Hardcover: not listened to recently")
		case last.AutoStartSkippedAt > s.now().Add(-autoStartRetry).UnixMilli():
			log.Debug("not starting book on Hardcover: it couldn't be started within the last day")
		default:
			audiobook, err = s.startReading(ctx, log, book, progress.Duration, r.userID)
			if err != nil {
				log.Error("starting book on Hardcover failed", "error", err)
				return failed
			}
			if audiobook == nil {
				entry := s.state.Books[progress.LibraryItemID]
				entry.AutoStartSkippedAt = s.now().UnixMilli()
				s.state.Books[progress.LibraryItemID] = entry
			}
			matchedBy = "started on Hardcover"
		}
	}
	if audiobook == nil {
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
	if read == nil && !s.autoStart {
		log.Error("book has no read on Hardcover")
		return failed
	}

	finishedAt := formatDate(progress.FinishedAt)
	// A read started after the book was finished is a re-read; don't mark it as read.
	if read != nil && progress.IsFinished && finishedAt != "" && read.StartedAt > finishedAt {
		log.Debug("Hardcover read started after the book was finished, treating it as a re-read",
			"readStartedAt", read.StartedAt, "finishedAt", finishedAt)
		return skipped
	}

	seconds := hardcoverSeconds(progress, audiobook.AudioSeconds)
	if seconds != listened {
		log.Debug("scaled progress to the Hardcover edition's length",
			"seconds", listened, "durationSeconds", int(math.Round(progress.Duration)),
			"hardcoverSeconds", seconds, "hardcoverDurationSeconds", audiobook.AudioSeconds)
	}

	update := hardcover.ReadUpdate{
		EditionID:       audiobook.EditionID,
		ProgressSeconds: seconds,
		StartedAt:       formatDate(progress.StartedAt),
		FinishedAt:      finishedAt,
	}
	switch {
	case read == nil:
		if update.StartedAt == "" {
			update.StartedAt = s.now().Format(time.DateOnly)
		}
		if _, err := s.hc.InsertRead(ctx, audiobook.UserBookID, update); err != nil {
			log.Error("adding a read on Hardcover failed", "error", err)
			return failed
		}
		log.Info("added a read with progress", "seconds", seconds)
	case read.ProgressSeconds == seconds && !progress.IsFinished:
		log.Info("progress unchanged", "seconds", seconds)
		s.remember(progress.LibraryItemID, audiobook.UserBookID, listened)
		return skipped
	default:
		if err := s.hc.UpdateRead(ctx, read.ID, update); err != nil {
			log.Error("updating progress on Hardcover failed", "error", err)
			return failed
		}
		log.Info("updated progress", "seconds", seconds)
	}
	s.remember(progress.LibraryItemID, audiobook.UserBookID, listened)

	if progress.IsFinished {
		if err := s.hc.MarkAsRead(ctx, audiobook.UserBookID, audiobook.EditionID); err != nil {
			log.Warn("marking book as read on Hardcover failed", "error", err)
		} else {
			log.Info("marked book as read")
			entry := s.state.Books[progress.LibraryItemID]
			entry.MarkedReadAt = progress.FinishedAt
			s.state.Books[progress.LibraryItemID] = entry
		}
	}
	return updated
}

// remember records the listening position synced to a user book.
func (s *Syncer) remember(libraryItemID string, userBookID, seconds int) {
	entry := s.state.Books[libraryItemID]
	entry.UserBookID = userBookID
	entry.Seconds = seconds
	s.state.Books[libraryItemID] = entry
}

// startReading puts a book on Hardcover as Currently Reading in its audiobook
// edition. It returns nil, with the reason logged, when there is no such
// edition or the book has a status that shouldn't change.
func (s *Syncer) startReading(ctx context.Context, log *slog.Logger, book audiobookshelf.Book, duration float64, userID int) (*hardcover.Audiobook, error) {
	edition, err := s.findAudioEdition(ctx, log, book, duration)
	if err != nil || edition == nil {
		return nil, err
	}
	log = log.With("editionId", edition.ID)

	userBook, err := s.hc.UserBook(ctx, userID, edition.BookID)
	if err != nil {
		return nil, fmt.Errorf("fetching book from Hardcover shelf: %w", err)
	}
	var userBookID int
	switch {
	case userBook == nil:
		userBookID, err = s.hc.AddCurrentlyReading(ctx, edition.BookID, edition.ID)
		if err != nil {
			return nil, fmt.Errorf("adding book to Hardcover: %w", err)
		}
		if userBookID == 0 {
			return nil, errors.New("adding book to Hardcover: no user book ID returned")
		}
		log.Info("added book to Hardcover as currently reading")
	case userBook.StatusID == hardcover.StatusWantToRead || userBook.StatusID == hardcover.StatusCurrentlyReading:
		userBookID = userBook.ID
		if err := s.hc.StartReading(ctx, userBookID, edition.ID); err != nil {
			return nil, fmt.Errorf("starting book on Hardcover: %w", err)
		}
		if userBook.StatusID == hardcover.StatusWantToRead {
			log.Info("changed book from want to read to currently reading on Hardcover")
		} else {
			log.Info("switched book to its audiobook edition on Hardcover")
		}
	default:
		// Read, paused, did not finish or ignored: the user decided, so leave it.
		log.Info("not starting book on Hardcover: it has another status there", "statusId", userBook.StatusID)
		return nil, nil
	}

	return &hardcover.Audiobook{
		UserBookID:   userBookID,
		EditionID:    edition.ID,
		Title:        edition.Title,
		AudioSeconds: edition.AudioSeconds,
	}, nil
}

// findAudioEdition looks up a book's audiobook edition on Hardcover by ISBN or
// ASIN, then by searching for its title and author. duration is the length of
// the Audiobookshelf file, used to pick between editions. It returns nil, with
// the reason logged, when there is none.
func (s *Syncer) findAudioEdition(ctx context.Context, log *slog.Logger, book audiobookshelf.Book, duration float64) (*hardcover.Edition, error) {
	isbn := normalizeISBN(book.ISBN)
	asin := strings.ToUpper(strings.TrimSpace(book.ASIN))
	if isbn != "" || asin != "" {
		edition, err := s.hc.FindAudioEdition(ctx, isbn, asin)
		if err != nil {
			return nil, fmt.Errorf("finding audiobook edition: %w", err)
		}
		if edition != nil {
			log.Debug("found audiobook edition on Hardcover", "by", "ISBN or ASIN")
			return edition, nil
		}
	}

	if len(book.Authors) == 0 {
		log.Info("not starting book on Hardcover: no audiobook edition with its ISBN or ASIN, and no author to search by",
			"isbn", isbn, "asin", asin)
		return nil, nil
	}

	query := book.Title + " " + book.Authors[0]
	ids, err := s.hc.SearchBooks(ctx, query, searchResults)
	if err != nil {
		return nil, fmt.Errorf("searching Hardcover: %w", err)
	}
	candidates, err := s.hc.CatalogBooks(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetching search results from Hardcover: %w", err)
	}

	var withoutAudio *hardcover.CatalogBook
	for i, candidate := range candidates {
		if !sharesAuthor(book.Authors, candidate.Authors) || !catalogTitleMatches(book.Title, candidate) {
			continue
		}
		if len(candidate.AudioEditions) == 0 {
			if withoutAudio == nil {
				withoutAudio = &candidates[i]
			}
			continue
		}
		edition := pickAudioEdition(book.Title, duration, candidate)
		log.Debug("found audiobook edition on Hardcover", "by", "title and author",
			"hardcoverBookId", candidate.ID, "editionTitle", edition.Title, "editionSeconds", edition.AudioSeconds)
		return &edition, nil
	}

	if withoutAudio != nil {
		log.Info("not starting book on Hardcover: the book is there but has no audiobook edition; add one on Hardcover",
			"hardcoverBookId", withoutAudio.ID, "hardcoverTitle", withoutAudio.Title)
		return nil, nil
	}
	log.Info("not starting book on Hardcover: not found by ISBN, ASIN, or title and author",
		"isbn", isbn, "asin", asin, "search", query)
	return nil, nil
}

// catalogTitleMatches reports whether a title matches a catalog book's title,
// one of its alternative titles, or one of its audio editions' titles, which
// may be in another language.
func catalogTitleMatches(title string, book hardcover.CatalogBook) bool {
	candidates := append([]string{book.Title}, book.AlternativeTitles...)
	for _, edition := range book.AudioEditions {
		candidates = append(candidates, edition.Title)
	}
	for _, candidate := range candidates {
		if looseTitlesMatch(title, candidate) {
			return true
		}
	}
	return false
}

// pickAudioEdition returns the audio edition whose length is closest to
// duration, among those whose title matches, so a translation gets its own
// edition and an abridged or dramatised one isn't mistaken for the full
// reading. Without a known length it returns the most popular one.
func pickAudioEdition(title string, duration float64, book hardcover.CatalogBook) hardcover.Edition {
	var matching []hardcover.Edition
	for _, candidate := range book.AudioEditions {
		if looseTitlesMatch(title, candidate.Title) {
			matching = append(matching, candidate)
		}
	}
	if len(matching) == 0 {
		matching = book.AudioEditions
	}

	edition := matching[0]
	if duration > 0 {
		best := math.Inf(1)
		for _, candidate := range matching {
			if candidate.AudioSeconds <= 0 {
				continue
			}
			if diff := math.Abs(float64(candidate.AudioSeconds) - duration); diff < best {
				best, edition = diff, candidate
			}
		}
	}
	if edition.BookID == 0 {
		edition.BookID = book.ID
	}
	return edition
}

func (r *run) currentlyReading(userBookID int) bool {
	for _, ab := range r.audiobooks {
		if ab.UserBookID == userBookID {
			return true
		}
	}
	return false
}

// hardcoverSeconds returns the listening position to write to Hardcover. When
// the Hardcover edition's length differs noticeably from the Audiobookshelf
// file's, the position is scaled so the percentage matches; finished books are
// set to the edition's full length.
func hardcoverSeconds(progress audiobookshelf.MediaProgress, audioSeconds int) int {
	if audioSeconds > 0 {
		if progress.IsFinished {
			return audioSeconds
		}
		if progress.Duration > 0 && math.Abs(float64(audioSeconds)/progress.Duration-1) > durationTolerance {
			return int(math.Round(min(progress.CurrentTime/progress.Duration, 1) * float64(audioSeconds)))
		}
	}
	return int(math.Round(progress.CurrentTime))
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
