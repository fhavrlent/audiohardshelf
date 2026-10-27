package audiobookshelf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
	"time"
)

func TestCurrentlyListening(t *testing.T) {
	now := time.Now()
	recent := now.Add(-24 * time.Hour).UnixMilli()
	old := now.Add(-90 * 24 * time.Hour).UnixMilli()

	// 60 in-progress items: more than one page.
	var inProgress []map[string]string
	for i := range 60 {
		inProgress = append(inProgress, map[string]string{"id": fmt.Sprintf("item%d", i)})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/libraries", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{"libraries": []map[string]string{
			{"id": "books", "mediaType": "book"},
			{"id": "podcasts", "mediaType": "podcast"},
		}})
	})
	mux.HandleFunc("GET /api/libraries/{id}/items", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") != "books" {
			t.Errorf("fetched items of library %q", r.PathValue("id"))
		}
		if got := r.URL.Query().Get("filter"); got != "progress.aW4tcHJvZ3Jlc3M=" {
			t.Errorf("filter = %q", got)
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		start, end := min(page*limit, len(inProgress)), min((page+1)*limit, len(inProgress))
		writeJSON(w, map[string]any{"results": inProgress[start:end], "total": len(inProgress)})
	})
	mux.HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"mediaProgress": []map[string]any{
			{"libraryItemId": "item0", "mediaItemType": "book", "currentTime": 1},
			{"libraryItemId": "item59", "mediaItemType": "book", "currentTime": 2},
			{"libraryItemId": "stale", "mediaItemType": "book", "currentTime": 3},
			{"libraryItemId": "finished-recently", "mediaItemType": "book", "isFinished": true, "finishedAt": recent},
			{"libraryItemId": "finished-long-ago", "mediaItemType": "book", "isFinished": true, "finishedAt": old},
			{"libraryItemId": "item1", "mediaItemType": "podcast", "currentTime": 4},
		}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	progress, err := New(server.URL, "key", time.Second).CurrentlyListening(t.Context(), now.Add(-30*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	var ids []string
	for _, p := range progress {
		ids = append(ids, p.LibraryItemID)
	}
	want := []string{"item0", "item59", "finished-recently"}
	if !slices.Equal(ids, want) {
		t.Errorf("got %v, want %v", ids, want)
	}
}

func TestBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/items/li_1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]any{"media": map[string]any{"metadata": map[string]any{
			"title": "Dune", "isbn": "9780441013593", "asin": nil,
		}}})
	}))
	defer server.Close()

	book, err := New(server.URL, "key", time.Second).Book(t.Context(), "li_1")
	if err != nil {
		t.Fatal(err)
	}
	if want := (Book{Title: "Dune", ISBN: "9780441013593"}); book != want {
		t.Errorf("got %+v, want %+v", book, want)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
