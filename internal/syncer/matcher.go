package syncer

import (
	"strings"

	"github.com/fhavrlent/audiohardshelf/internal/audiobookshelf"
	"github.com/fhavrlent/audiohardshelf/internal/hardcover"
)

// findMatch returns the Hardcover audiobook matching an Audiobookshelf book,
// trying ISBN, then ASIN, then exact (case-insensitive) title, and what it matched on.
func findMatch(book audiobookshelf.Book, audiobooks []hardcover.Audiobook) (*hardcover.Audiobook, string) {
	if isbn := normalizeISBN(book.ISBN); isbn != "" {
		for i, ab := range audiobooks {
			if normalizeISBN(ab.ISBN10) == isbn || normalizeISBN(ab.ISBN13) == isbn {
				return &audiobooks[i], "ISBN"
			}
		}
	}

	if asin := strings.TrimSpace(book.ASIN); asin != "" {
		for i, ab := range audiobooks {
			if strings.EqualFold(strings.TrimSpace(ab.ASIN), asin) {
				return &audiobooks[i], "ASIN"
			}
		}
	}

	if title := strings.TrimSpace(book.Title); title != "" {
		for i, ab := range audiobooks {
			if strings.EqualFold(strings.TrimSpace(ab.Title), title) {
				return &audiobooks[i], "title"
			}
		}
	}

	return nil, ""
}

func normalizeISBN(isbn string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(isbn)
}
