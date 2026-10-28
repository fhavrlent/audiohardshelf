// Package state remembers what was synced between runs, so unchanged books
// can be skipped and finished books are marked as read only once.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Book is what was last synced for one Audiobookshelf library item.
type Book struct {
	// UserBookID is the Hardcover user book the item was synced to.
	UserBookID int `json:"userBookId,omitempty"`
	// Seconds is the Audiobookshelf listening position last synced.
	Seconds int `json:"seconds,omitempty"`
	// MarkedReadAt is the Audiobookshelf finish time, in epoch milliseconds,
	// of the listen that was marked as read on Hardcover.
	MarkedReadAt int64 `json:"markedReadAt,omitempty"`
}

// Store holds the state in memory and, when it has a path, in a JSON file.
type Store struct {
	path string

	// LastSync is when the last completed sync started; zero if there was none.
	LastSync time.Time `json:"lastSync"`
	// Books is keyed by Audiobookshelf library item ID.
	Books map[string]Book `json:"books"`
}

// Memory returns a store that is never saved.
func Memory() *Store {
	return &Store{Books: map[string]Book{}}
}

// Load reads the store from path. A missing file gives an empty store. When the
// file can't be read, it returns an error along with an empty store that
// replaces the file on the next save.
func Load(path string) (*Store, error) {
	fresh := Memory()
	fresh.path = path
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fresh, nil
	}
	if err != nil {
		return fresh, fmt.Errorf("reading state file: %w", err)
	}
	store := Memory()
	store.path = path
	if err := json.Unmarshal(data, store); err != nil {
		return fresh, fmt.Errorf("reading state file %s: %w", path, err)
	}
	if store.Books == nil {
		store.Books = map[string]Book{}
	}
	return store, nil
}

// Save writes the store to its file, replacing it atomically. Stores without
// a path are not saved.
func (s *Store) Save() error {
	if s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".state-*.json")
	if err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}
	return nil
}
