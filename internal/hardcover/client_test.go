package hardcover

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// serve answers every GraphQL request with response and records the last request.
func serve(t *testing.T, response string) (*Client, *map[string]any) {
	t.Helper()
	var last map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Errorf("Authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&last); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return New(server.URL, "Bearer key", time.Second), &last
}

func TestMe(t *testing.T) {
	for _, response := range []string{
		`{"data":{"me":[{"id":7,"username":"reader"}]}}`,
		`{"data":{"me":{"id":7,"username":"reader"}}}`,
	} {
		client, _ := serve(t, response)
		user, err := client.Me(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if user.ID != 7 || user.Username != "reader" {
			t.Errorf("got %+v from %s", user, response)
		}
	}
}

func TestGraphQLErrors(t *testing.T) {
	client, _ := serve(t, `{"errors":[{"message":"bad"},{"message":"worse"}]}`)
	_, err := client.Me(t.Context())
	if err == nil || !strings.Contains(err.Error(), "bad; worse") {
		t.Errorf("got error %v", err)
	}
}

func TestCurrentlyReadingAudiobooks(t *testing.T) {
	client, _ := serve(t, `{"data":{"user_books":[
		{"id":1,"edition_id":10,"book":{"title":"Book","contributions":[{"author":{"name":"Author"}},{"author":null}],
			"editions":[{"id":10,"isbn_13":"978","asin":"B01","title":null,"audio_seconds":3600}]}},
		{"id":2,"edition_id":20,"book":{"title":"Print edition","editions":[{"id":21}]}},
		{"id":3,"edition_id":null,"book":{"title":"No edition","editions":[{"id":30}]}}
	]}}`)

	audiobooks, err := client.CurrentlyReadingAudiobooks(t.Context(), 7)
	if err != nil {
		t.Fatal(err)
	}
	want := Audiobook{
		UserBookID: 1, EditionID: 10, Title: "Book", BookTitle: "Book", ISBN13: "978", ASIN: "B01",
		AudioSeconds: 3600, Authors: []string{"Author"},
	}
	if len(audiobooks) != 1 || !reflect.DeepEqual(audiobooks[0], want) {
		t.Errorf("got %+v, want [%+v]", audiobooks, want)
	}
}

func TestMutationError(t *testing.T) {
	client, last := serve(t, `{"data":{"result":{"error":"not allowed"}}}`)
	err := client.UpdateRead(t.Context(), 5, ReadUpdate{EditionID: 10, ProgressSeconds: 60})
	if err == nil || err.Error() != "not allowed" {
		t.Errorf("got error %v", err)
	}

	object := (*last)["variables"].(map[string]any)["object"].(map[string]any)
	if _, ok := object["started_at"]; ok {
		t.Errorf("empty started_at was sent: %v", object)
	}
	if object["progress_seconds"] != float64(60) {
		t.Errorf("progress_seconds = %v", object["progress_seconds"])
	}
}

func TestFindAudioEdition(t *testing.T) {
	client, last := serve(t, `{"data":{"editions":[{"id":10,"book_id":1,"title":"Dune","audio_seconds":75600}]}}`)
	edition, err := client.FindAudioEdition(t.Context(), "9780441013593", "B002V0QK4C")
	if err != nil {
		t.Fatal(err)
	}
	if want := (Edition{ID: 10, BookID: 1, Title: "Dune", AudioSeconds: 75600}); edition == nil || *edition != want {
		t.Errorf("got %+v, want %+v", edition, want)
	}
	where := (*last)["variables"].(map[string]any)["where"].(map[string]any)
	if _, ok := where["isbn_13"]; !ok || where["asin"] != nil || where["_or"] == nil {
		t.Errorf("searched with %v, want ISBN-13 and audio editions only", where)
	}
}

func TestFindAudioEditionNone(t *testing.T) {
	client, last := serve(t, `{"data":{"editions":[]}}`)
	edition, err := client.FindAudioEdition(t.Context(), "", "B002V0QK4C")
	if err != nil || edition != nil {
		t.Fatalf("got %+v, %v; want nil", edition, err)
	}
	where := (*last)["variables"].(map[string]any)["where"].(map[string]any)
	if _, ok := where["asin"]; !ok {
		t.Errorf("searched with %v, want ASIN", where)
	}

	edition, err = client.FindAudioEdition(t.Context(), "", "")
	if err != nil || edition != nil {
		t.Errorf("got %+v, %v without identifiers; want nil", edition, err)
	}
}

func TestAddCurrentlyReading(t *testing.T) {
	client, last := serve(t, `{"data":{"result":{"id":42,"error":null}}}`)
	id, err := client.AddCurrentlyReading(t.Context(), 1, 10)
	if err != nil || id != 42 {
		t.Fatalf("got %d, %v; want 42", id, err)
	}
	object := (*last)["variables"].(map[string]any)["object"].(map[string]any)
	if object["book_id"] != float64(1) || object["edition_id"] != float64(10) || object["status_id"] != float64(StatusCurrentlyReading) {
		t.Errorf("sent %v", object)
	}
}

func TestUserBookNotOnShelf(t *testing.T) {
	client, _ := serve(t, `{"data":{"user_books":[]}}`)
	userBook, err := client.UserBook(t.Context(), 7, 1)
	if err != nil || userBook != nil {
		t.Errorf("got %+v, %v; want nil", userBook, err)
	}
}

func TestSearchBooks(t *testing.T) {
	for _, response := range []string{
		`{"data":{"search":{"ids":[3,1]}}}`,
		`{"data":{"search":{"ids":[],"results":{"hits":[{"document":{"id":"3"}},{"document":{"id":"1"}}]}}}}`,
	} {
		client, last := serve(t, response)
		ids, err := client.SearchBooks(t.Context(), "Dune Frank Herbert", 10)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ids, []int{3, 1}) {
			t.Errorf("got %v from %s, want [3 1]", ids, response)
		}
		if got := (*last)["variables"].(map[string]any)["query"]; got != "Dune Frank Herbert" {
			t.Errorf("searched for %v", got)
		}
	}
}

func TestCatalogBooks(t *testing.T) {
	client, _ := serve(t, `{"data":{"books":[
		{"id":1,"title":"Night Watch","alternative_titles":["Ночной дозор"],"contributions":[{"author":{"name":"Sergei Lukyanenko"}}],
			"editions":[{"id":10,"book_id":1,"title":"Noční hlídka","audio_seconds":50000}]},
		{"id":3,"title":"Print only","alternative_titles":null,"contributions":[],"editions":[]}
	]}}`)
	books, err := client.CatalogBooks(t.Context(), []int{3, 2, 1})
	if err != nil {
		t.Fatal(err)
	}
	want := []CatalogBook{
		{ID: 3, Title: "Print only", AudioEditions: []Edition{}},
		{ID: 1, Title: "Night Watch", AlternativeTitles: []string{"Ночной дозор"}, Authors: []string{"Sergei Lukyanenko"},
			AudioEditions: []Edition{{ID: 10, BookID: 1, Title: "Noční hlídka", AudioSeconds: 50000}}},
	}
	if !reflect.DeepEqual(books, want) {
		t.Errorf("got %+v, want %+v in search order", books, want)
	}
}
