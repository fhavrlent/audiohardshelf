package syncer

import (
	"testing"

	"github.com/fhavrlent/audiohardshelf/internal/audiobookshelf"
	"github.com/fhavrlent/audiohardshelf/internal/hardcover"
)

func TestFindMatch(t *testing.T) {
	audiobooks := []hardcover.Audiobook{
		{EditionID: 1, Title: "By Title"},
		{EditionID: 2, Title: "Other", ISBN10: "0441013597"},
		{EditionID: 3, Title: "Other", ASIN: "B002V0QK4C"},
		{EditionID: 4, Title: ""},
	}

	tests := []struct {
		name    string
		book    audiobookshelf.Book
		edition int
		by      string
	}{
		{"isbn with hyphens", audiobookshelf.Book{ISBN: "0-441-01359-7", Title: "By Title"}, 2, "ISBN"},
		{"asin ignores case", audiobookshelf.Book{ASIN: "b002v0qk4c"}, 3, "ASIN"},
		{"isbn miss falls back to title", audiobookshelf.Book{ISBN: "123", Title: " by title "}, 1, "title"},
		{"empty title never matches", audiobookshelf.Book{}, 0, ""},
		{"no match", audiobookshelf.Book{Title: "Missing"}, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, by := findMatch(tt.book, audiobooks)
			edition := 0
			if match != nil {
				edition = match.EditionID
			}
			if edition != tt.edition || by != tt.by {
				t.Errorf("matched edition %d by %q, want %d by %q", edition, by, tt.edition, tt.by)
			}
		})
	}
}
