package hardcover

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		{"id":1,"edition_id":10,"book":{"title":"Book","editions":[{"id":10,"isbn_13":"978","asin":"B01","title":null}]}},
		{"id":2,"edition_id":20,"book":{"title":"Print edition","editions":[{"id":21}]}},
		{"id":3,"edition_id":null,"book":{"title":"No edition","editions":[{"id":30}]}}
	]}}`)

	audiobooks, err := client.CurrentlyReadingAudiobooks(t.Context(), 7)
	if err != nil {
		t.Fatal(err)
	}
	want := Audiobook{UserBookID: 1, EditionID: 10, Title: "Book", ISBN13: "978", ASIN: "B01"}
	if len(audiobooks) != 1 || audiobooks[0] != want {
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
