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
		{EditionID: 5, Title: "Dune: Book One", Authors: []string{"Frank Herbert"}},
		{EditionID: 6, Title: "Mistborn: The Well of Ascension", Authors: []string{"Brandon Sanderson"}},
		{EditionID: 7, Title: "Audible Edition", BookTitle: "The Fellowship of the Ring (The Lord of the Rings, #1)", Authors: []string{"J. R. R. Tolkien"}},
		{EditionID: 8, Title: "Shared Title", Authors: []string{"Someone Else"}},
		{EditionID: 9, Title: "Shared Title", Authors: []string{"Right Author"}},
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
		{"title without subtitle", audiobookshelf.Book{Title: "Dune", Authors: []string{"frank herbert"}}, 5, "title and author"},
		{"loose title needs an author", audiobookshelf.Book{Title: "Dune"}, 0, ""},
		{"loose title needs the same author", audiobookshelf.Book{Title: "Dune", Authors: []string{"Brian Herbert"}}, 0, ""},
		{"different subtitles", audiobookshelf.Book{Title: "Mistborn: The Final Empire", Authors: []string{"Brandon Sanderson"}}, 0, ""},
		{"book title, article and series", audiobookshelf.Book{Title: "Fellowship of the Ring", Authors: []string{"J.R.R. Tolkien"}}, 7, "title and author"},
		{"exact title skips other authors", audiobookshelf.Book{Title: "Shared Title", Authors: []string{"Right Author"}}, 9, "title"},
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
