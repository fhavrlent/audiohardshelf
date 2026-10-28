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
	"github.com/fhavrlent/audiohardshelf/internal/state"
)

type fakeABS struct {
	progress      []audiobookshelf.MediaProgress
	books         map[string]audiobookshelf.Book
	finishedSince time.Time
}

func (f *fakeABS) Ping(context.Context) error { return nil }

func (f *fakeABS) CurrentlyListening(_ context.Context, finishedSince time.Time) ([]audiobookshelf.MediaProgress, error) {
	f.finishedSince = finishedSince
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
	readsFetch int

	// Catalog and shelf for auto-start, keyed by ISBN or ASIN and by book ID.
	editions    map[string]hardcover.Edition
	userBooks   map[int]hardcover.UserBook
	added       []int // book IDs
	started     []int // user book IDs
	insertedFor map[int]hardcover.ReadUpdate
	// Search results by query, and catalog books by ID.
	search   map[string][]int
	catalog  map[int]hardcover.CatalogBook
	searches []string
}

func (f *fakeHardcover) Me(context.Context) (hardcover.User, error) {
	return hardcover.User{ID: 1, Username: "reader"}, nil
}

func (f *fakeHardcover) CurrentlyReadingAudiobooks(context.Context, int) ([]hardcover.Audiobook, error) {
	return f.audiobooks, nil
}

func (f *fakeHardcover) LatestRead(_ context.Context, userBookID int) (*hardcover.Read, error) {
	f.readsFetch++
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

func (f *fakeHardcover) FindAudioEdition(_ context.Context, isbn, asin string) (*hardcover.Edition, error) {
	for _, id := range []string{isbn, asin} {
		if edition, ok := f.editions[id]; ok && id != "" {
			return &edition, nil
		}
	}
	return nil, nil
}

func (f *fakeHardcover) SearchBooks(_ context.Context, query string, _ int) ([]int, error) {
	f.searches = append(f.searches, query)
	return f.search[query], nil
}

func (f *fakeHardcover) CatalogBooks(_ context.Context, ids []int) ([]hardcover.CatalogBook, error) {
	var books []hardcover.CatalogBook
	for _, id := range ids {
		books = append(books, f.catalog[id])
	}
	return books, nil
}

func (f *fakeHardcover) UserBook(_ context.Context, _, bookID int) (*hardcover.UserBook, error) {
	userBook, ok := f.userBooks[bookID]
	if !ok {
		return nil, nil
	}
	return &userBook, nil
}

func (f *fakeHardcover) AddCurrentlyReading(_ context.Context, bookID, _ int) (int, error) {
	f.added = append(f.added, bookID)
	return 1000 + bookID, nil
}

func (f *fakeHardcover) StartReading(_ context.Context, userBookID, _ int) error {
	f.started = append(f.started, userBookID)
	return nil
}

func (f *fakeHardcover) InsertRead(_ context.Context, userBookID int, read hardcover.ReadUpdate) (int, error) {
	f.insertedFor[userBookID] = read
	return 1, nil
}

func discard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
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

	s := New(abs, hc, discard(), Options{})
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

func TestRunRemembersSyncs(t *testing.T) {
	abs := &fakeABS{
		progress: []audiobookshelf.MediaProgress{
			{LibraryItemID: "listening", MediaItemType: "book", CurrentTime: 100},
			{LibraryItemID: "finished", MediaItemType: "book", CurrentTime: 500, IsFinished: true, FinishedAt: ms("2026-10-20")},
		},
		books: map[string]audiobookshelf.Book{
			"listening": {Title: "Listening"},
			"finished":  {Title: "Finished"},
		},
	}
	hc := &fakeHardcover{
		audiobooks: []hardcover.Audiobook{
			{UserBookID: 1, EditionID: 10, Title: "Listening"},
			{UserBookID: 2, EditionID: 20, Title: "Finished"},
		},
		reads: map[int]hardcover.Read{
			1: {ID: 101, ProgressSeconds: 50},
			2: {ID: 102, ProgressSeconds: 400},
		},
		updates: map[int]hardcover.ReadUpdate{},
	}
	store := state.Memory()
	s := New(abs, hc, discard(), Options{State: store})
	s.now = func() time.Time { return time.UnixMilli(ms("2026-10-21")) }

	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(hc.updates) != 2 || !slices.Equal(hc.markedRead, []int{2}) {
		t.Fatalf("first sync updated %v and marked %v as read", hc.updates, hc.markedRead)
	}

	// The same day, a re-read is started on Hardcover; it is still Currently Reading.
	hc.reads[2] = hardcover.Read{ID: 103, StartedAt: "2026-10-20"}
	hc.updates = map[int]hardcover.ReadUpdate{}
	hc.readsFetch = 0
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(hc.updates) != 0 || hc.readsFetch != 0 || len(hc.markedRead) != 1 {
		t.Errorf("second sync updated %v, fetched %d reads and marked %v as read; want nothing",
			hc.updates, hc.readsFetch, hc.markedRead)
	}

	// Listening moves on; books that are gone are forgotten.
	abs.progress = []audiobookshelf.MediaProgress{{LibraryItemID: "listening", MediaItemType: "book", CurrentTime: 200}}
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if hc.updates[101].ProgressSeconds != 200 {
		t.Errorf("third sync updated %v, want progress 200", hc.updates)
	}
	if _, ok := store.Books["finished"]; ok || len(store.Books) != 1 {
		t.Errorf("state kept %v", store.Books)
	}
}

func TestRunLooksBackToLastSync(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name     string
		lastSync time.Time
		want     time.Time
	}{
		{"first sync", time.Time{}, now.Add(-finishedLookback)},
		{"recent sync", now.Add(-time.Hour), now.Add(-finishedLookback)},
		{"sync long ago", now.Add(-60 * 24 * time.Hour), now.Add(-61 * 24 * time.Hour)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			abs := &fakeABS{}
			store := state.Memory()
			store.LastSync = tt.lastSync
			s := New(abs, &fakeHardcover{}, discard(), Options{State: store})
			s.now = func() time.Time { return now }
			if err := s.Run(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !abs.finishedSince.Equal(tt.want) {
				t.Errorf("looked back to %v, want %v", abs.finishedSince, tt.want)
			}
			if !store.LastSync.Equal(now) {
				t.Errorf("last sync = %v, want %v", store.LastSync, now)
			}
		})
	}
}

func TestRunAutoStart(t *testing.T) {
	now := time.UnixMilli(ms("2026-10-21"))
	recent := now.Add(-time.Hour).UnixMilli()
	abs := &fakeABS{
		progress: []audiobookshelf.MediaProgress{
			// Not on the shelf: added, and a read is created.
			{LibraryItemID: "new", MediaItemType: "book", CurrentTime: 60, StartedAt: ms("2026-10-19"), LastUpdate: recent},
			// Want to read: started, and its existing read updated.
			{LibraryItemID: "wanted", MediaItemType: "book", CurrentTime: 70, LastUpdate: recent},
			// Did not finish on Hardcover: left alone.
			{LibraryItemID: "dnf", MediaItemType: "book", CurrentTime: 80, LastUpdate: recent},
			// Not listened to lately: not started.
			{LibraryItemID: "stale", MediaItemType: "book", CurrentTime: 90, LastUpdate: now.Add(-30 * 24 * time.Hour).UnixMilli()},
			// No audiobook edition with this ASIN.
			{LibraryItemID: "missing", MediaItemType: "book", CurrentTime: 95, LastUpdate: recent},
		},
		books: map[string]audiobookshelf.Book{
			"new":     {Title: "New", ISBN: "978-1", ASIN: "b1"},
			"wanted":  {Title: "Wanted", ASIN: "B2"},
			"dnf":     {Title: "DNF", ASIN: "B3"},
			"stale":   {Title: "Stale", ASIN: "B4"},
			"missing": {Title: "Missing", ASIN: "B5"},
		},
	}
	hc := &fakeHardcover{
		editions: map[string]hardcover.Edition{
			"B1": {ID: 10, BookID: 1, AudioSeconds: 1000},
			"B2": {ID: 20, BookID: 2},
			"B3": {ID: 30, BookID: 3},
			"B4": {ID: 40, BookID: 4},
		},
		userBooks: map[int]hardcover.UserBook{
			2: {ID: 2002, StatusID: hardcover.StatusWantToRead},
			3: {ID: 2003, StatusID: 5},
		},
		reads:       map[int]hardcover.Read{2002: {ID: 202}},
		updates:     map[int]hardcover.ReadUpdate{},
		insertedFor: map[int]hardcover.ReadUpdate{},
	}

	s := New(abs, hc, discard(), Options{AutoStart: true})
	s.now = func() time.Time { return now }
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(hc.added, []int{1}) || !slices.Equal(hc.started, []int{2002}) {
		t.Errorf("added books %v and started user books %v, want [1] and [2002]", hc.added, hc.started)
	}
	if want := (hardcover.ReadUpdate{EditionID: 10, ProgressSeconds: 60, StartedAt: "2026-10-19"}); hc.insertedFor[1001] != want || len(hc.insertedFor) != 1 {
		t.Errorf("inserted reads %v, want %+v for user book 1001", hc.insertedFor, want)
	}
	if want := (hardcover.ReadUpdate{EditionID: 20, ProgressSeconds: 70}); hc.updates[202] != want || len(hc.updates) != 1 {
		t.Errorf("updated reads %v, want %+v for read 202", hc.updates, want)
	}
}

func TestRunAutoStartByTitle(t *testing.T) {
	now := time.UnixMilli(ms("2026-10-21"))
	recent := now.Add(-time.Hour).UnixMilli()
	abs := &fakeABS{
		progress: []audiobookshelf.MediaProgress{
			// No ISBN or ASIN: found by title and author.
			{LibraryItemID: "czech", MediaItemType: "book", CurrentTime: 60, LastUpdate: recent},
			// A translation: the book's title is the original, its Czech audio edition matches.
			{LibraryItemID: "translated", MediaItemType: "book", CurrentTime: 70, LastUpdate: recent},
			// On Hardcover, but only in print.
			{LibraryItemID: "print", MediaItemType: "book", CurrentTime: 80, LastUpdate: recent},
			// No author to search by.
			{LibraryItemID: "anonymous", MediaItemType: "book", CurrentTime: 90, LastUpdate: recent},
		},
		books: map[string]audiobookshelf.Book{
			"czech":      {Title: "Aristokratka v ohrožení", Authors: []string{"Evžen Boček"}},
			"translated": {Title: "Noční hlídka", Authors: []string{"Sergej Lukjaněnko"}},
			"print":      {Title: "Jen tištěná", Authors: []string{"Autor"}},
			"anonymous":  {Title: "Bez autora"},
		},
	}
	hc := &fakeHardcover{
		search: map[string][]int{
			"Aristokratka v ohrožení Evžen Boček": {9, 1},
			"Noční hlídka Sergej Lukjaněnko":      {2},
			"Jen tištěná Autor":                   {3},
		},
		catalog: map[int]hardcover.CatalogBook{
			// Same title, another author: skipped.
			9: {ID: 9, Title: "Aristokratka v ohrožení", Authors: []string{"Someone Else"},
				AudioEditions: []hardcover.Edition{{ID: 90}}},
			1: {ID: 1, Title: "Aristokratka v ohrožení", Authors: []string{"Evžen Boček"},
				AudioEditions: []hardcover.Edition{{ID: 10, Title: "Aristokratka v ohrožení"}}},
			2: {ID: 2, Title: "Night Watch", Authors: []string{"Sergej Lukjaněnko"},
				AudioEditions: []hardcover.Edition{{ID: 20, BookID: 2, Title: "Night Watch"}, {ID: 21, BookID: 2, Title: "Noční hlídka"}}},
			3: {ID: 3, Title: "Jen tištěná", Authors: []string{"Autor"}},
		},
		updates:     map[int]hardcover.ReadUpdate{},
		insertedFor: map[int]hardcover.ReadUpdate{},
	}

	s := New(abs, hc, discard(), Options{AutoStart: true})
	s.now = func() time.Time { return now }
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(hc.added, []int{1, 2}) {
		t.Errorf("added books %v, want [1 2]", hc.added)
	}
	if hc.insertedFor[1001].EditionID != 10 || hc.insertedFor[1002].EditionID != 21 {
		t.Errorf("inserted reads %v, want edition 10 for book 1 and the Czech edition 21 for book 2", hc.insertedFor)
	}
	if len(hc.searches) != 3 {
		t.Errorf("searched %q, want 3 searches and none without an author", hc.searches)
	}

	// Books that couldn't be started aren't looked up again the same day.
	hc.searches = nil
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(hc.searches, "Jen tištěná Autor") {
		t.Errorf("searched again for %q", hc.searches)
	}
	s.now = func() time.Time { return now.Add(25 * time.Hour) }
	abs.progress[2].LastUpdate = now.Add(24 * time.Hour).UnixMilli()
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(hc.searches, "Jen tištěná Autor") {
		t.Errorf("searched %q a day later, want the print-only book retried", hc.searches)
	}
}

func TestHardcoverSeconds(t *testing.T) {
	for _, tt := range []struct {
		name         string
		progress     audiobookshelf.MediaProgress
		audioSeconds int
		want         int
	}{
		{"unknown edition length", audiobookshelf.MediaProgress{CurrentTime: 99.6, Duration: 1000}, 0, 100},
		{"unknown file length", audiobookshelf.MediaProgress{CurrentTime: 100}, 2000, 100},
		{"lengths within tolerance", audiobookshelf.MediaProgress{CurrentTime: 500, Duration: 1000}, 1040, 500},
		{"longer edition", audiobookshelf.MediaProgress{CurrentTime: 500, Duration: 1000}, 2000, 1000},
		{"shorter edition", audiobookshelf.MediaProgress{CurrentTime: 500, Duration: 1000}, 800, 400},
		{"past the end", audiobookshelf.MediaProgress{CurrentTime: 1200, Duration: 1000}, 2000, 2000},
		{"finished", audiobookshelf.MediaProgress{CurrentTime: 0, Duration: 1000, IsFinished: true}, 1040, 1040},
		{"finished, unknown edition length", audiobookshelf.MediaProgress{CurrentTime: 500, IsFinished: true}, 0, 500},
	} {
		if got := hardcoverSeconds(tt.progress, tt.audioSeconds); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestPickAudioEdition(t *testing.T) {
	styles := hardcover.CatalogBook{ID: 1, Title: "The Mysterious Affair at Styles", AudioEditions: []hardcover.Edition{
		{ID: 10, Title: "The Mysterious Affair at Styles: An Audible Original Full Cast Audio Drama", AudioSeconds: 13980},
		{ID: 11, Title: "The Mysterious Affair at Styles: A Hercule Poirot Mystery", AudioSeconds: 21420},
		{ID: 12, Title: "Záhada na zámku Styles", AudioSeconds: 21600},
		{ID: 13, Title: "The Mysterious Affair at Styles"},
	}}
	tests := []struct {
		name     string
		title    string
		duration float64
		want     int
	}{
		{"closest length", "The Mysterious Affair at Styles", 21500, 11},
		{"shorter file", "The Mysterious Affair at Styles", 14000, 10},
		{"unknown length", "The Mysterious Affair at Styles", 0, 10},
		{"translation", "Záhada na zámku Styles", 14000, 12},
		{"no title match", "Something Else", 21700, 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickAudioEdition(tt.title, tt.duration, styles)
			if got.ID != tt.want || got.BookID != 1 {
				t.Errorf("picked edition %d of book %d, want %d of book 1", got.ID, got.BookID, tt.want)
			}
		})
	}
}
