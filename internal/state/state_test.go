package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	store, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !store.LastSync.IsZero() || len(store.Books) != 0 {
		t.Fatalf("missing file loaded as %+v", store)
	}

	store.LastSync = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	store.Books["li_1"] = Book{UserBookID: 7, Seconds: 120, MarkedReadAt: 1700000000000}
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.LastSync.Equal(store.LastSync) || loaded.Books["li_1"] != store.Books["li_1"] {
		t.Errorf("loaded %+v, want %+v", loaded, store)
	}

	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("temporary files left behind: %v", entries)
	}
}

func TestLoadInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Load(path)
	if err == nil {
		t.Error("loaded invalid JSON without an error")
	}
	if store == nil || store.Books == nil {
		t.Fatalf("got store %+v, want an empty one", store)
	}
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Errorf("saving did not replace the invalid file: %v", err)
	}
}

func TestMemoryIsNotSaved(t *testing.T) {
	if err := Memory().Save(); err != nil {
		t.Error(err)
	}
}
