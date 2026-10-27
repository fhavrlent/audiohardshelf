package syncer

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/fhavrlent/audiohardshelf/internal/audiobookshelf"
	"github.com/fhavrlent/audiohardshelf/internal/hardcover"
)

type fakeABS struct {
	progress []audiobookshelf.MediaProgress
	books    map[string]audiobookshelf.Book
}

func (f *fakeABS) Ping(context.Context) error { return nil }

func (f *fakeABS) CurrentlyListening(context.Context, time.Time) ([]audiobookshelf.MediaProgress, error) {
	return f.progress, nil
}

func (f *fakeABS) Book(_ context.Context, id string) (audiobookshelf.Book, error) {
	return f.books[id], nil
}

type fakeHardcover struct {
	audiobooks []hardcover.Audiobook
	reads      map[int]hardcover.Read
	updates    map[int]hardcover.ReadUpdate
	markedRead []int
}

func (f *fakeHardcover) Me(context.Context) (hardcover.User, error) {
	return hardcover.User{ID: 1, Username: "reader"}, nil
}

func (f *fakeHardcover) CurrentlyReadingAudiobooks(context.Context, int) ([]hardcover.Audiobook, error) {
	return f.audiobooks, nil
}

func (f *fakeHardcover) LatestRead(_ context.Context, userBookID int) (*hardcover.Read, error) {
	read, ok := f.reads[userBookID]
	if !ok {
		return nil, nil
	}
	return &read, nil
}

func (f *fakeHardcover) UpdateRead(_ context.Context, readID int, update hardcover.ReadUpdate) error {
	f.updates[readID] = update
	return nil
}

func (f *fakeHardcover) MarkAsRead(_ context.Context, userBookID, _ int) error {
	f.markedRead = append(f.markedRead, userBookID)
	return nil
}

func ms(date string) int64 {
	t, err := time.ParseInLocation(time.DateOnly, date, time.Local)
	if err != nil {
		panic(err)
	}
	return t.Add(12 * time.Hour).UnixMilli()
}

func TestRun(t *testing.T) {
	abs := &fakeABS{
		progress: []audiobookshelf.MediaProgress{
			// In progress, position changed.
			{LibraryItemID: "listening", MediaItemType: "book", CurrentTime: 99.6, StartedAt: ms("2026-10-01")},
			// Duplicate record is synced once.
			{LibraryItemID: "listening", MediaItemType: "book", CurrentTime: 99.6},
			// In progress, position unchanged.
			{LibraryItemID: "unchanged", MediaItemType: "book", CurrentTime: 300},
			// Finished at the position already on Hardcover: still marked as read.
			{LibraryItemID: "finished", MediaItemType: "book", CurrentTime: 500, IsFinished: true, FinishedAt: ms("2026-10-20")},
			// Finished, but Hardcover has since started a re-read.
			{LibraryItemID: "reread", MediaItemType: "book", CurrentTime: 700, IsFinished: true, FinishedAt: ms("2026-10-05")},
			// Finished and no longer currently reading on Hardcover.
			{LibraryItemID: "done", MediaItemType: "book", CurrentTime: 900, IsFinished: true, FinishedAt: ms("2026-10-20")},
			// Not on Hardcover.
			{LibraryItemID: "unknown", MediaItemType: "book", CurrentTime: 10},
			// Only the ebook was read: listening position 0 must not reset Hardcover progress.
			{LibraryItemID: "ebook", MediaItemType: "book", CurrentTime: 0},
		},
		books: map[string]audiobookshelf.Book{
			"listening": {Title: "Listening", ISBN: "978-0-00-000000-1"},
			"unchanged": {Title: "Unchanged", ASIN: "B000000002"},
			"finished":  {Title: "Finished"},
			"reread":    {Title: "Reread"},
			"done":      {Title: "Done"},
			"unknown":   {Title: "Unknown"},
			"ebook":     {Title: "Ebook"},
		},
	}
	hc := &fakeHardcover{
		audiobooks: []hardcover.Audiobook{
			{UserBookID: 1, EditionID: 10, Title: "Something else", ISBN13: "9780000000001"},
			{UserBookID: 2, EditionID: 20, Title: "Unchanged", ASIN: "b000000002"},
			{UserBookID: 3, EditionID: 30, Title: "finished"},
			{UserBookID: 4, EditionID: 40, Title: "Reread"},
			{UserBookID: 5, EditionID: 50, Title: "Ebook"},
		},
		reads: map[int]hardcover.Read{
			1: {ID: 101, ProgressSeconds: 50},
			2: {ID: 102, ProgressSeconds: 300},
			3: {ID: 103, ProgressSeconds: 500},
			4: {ID: 104, ProgressSeconds: 0, StartedAt: "2026-10-10"},
			5: {ID: 105, ProgressSeconds: 3000},
		},
		updates: map[int]hardcover.ReadUpdate{},
	}

	s := New(abs, hc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	want := map[int]hardcover.ReadUpdate{
		101: {EditionID: 10, ProgressSeconds: 100, StartedAt: "2026-10-01"},
		103: {EditionID: 30, ProgressSeconds: 500, FinishedAt: "2026-10-20"},
	}
	if len(hc.updates) != len(want) {
		t.Errorf("updated reads %v, want %v", hc.updates, want)
	}
	for id, update := range want {
		if hc.updates[id] != update {
			t.Errorf("read %d updated with %+v, want %+v", id, hc.updates[id], update)
		}
	}
	if !slices.Equal(hc.markedRead, []int{3}) {
		t.Errorf("marked as read %v, want [3]", hc.markedRead)
	}
}
