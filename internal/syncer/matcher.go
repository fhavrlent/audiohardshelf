package syncer

import (
	"strings"
	"unicode"

	"github.com/fhavrlent/audiohardshelf/internal/audiobookshelf"
	"github.com/fhavrlent/audiohardshelf/internal/hardcover"
)

// findMatch returns the Hardcover audiobook matching an Audiobookshelf book,
// and what it matched on. It tries ISBN, then ASIN, then the exact title
// (ignoring case), then a loose title that also needs a shared author.
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

	title := strings.TrimSpace(book.Title)
	if title == "" {
		return nil, ""
	}

	// An exact title is enough on its own, unless both sides list authors and none are shared.
	for i, ab := range audiobooks {
		for _, candidate := range titles(ab) {
			if strings.EqualFold(strings.TrimSpace(candidate), title) &&
				(len(book.Authors) == 0 || len(ab.Authors) == 0 || sharesAuthor(book.Authors, ab.Authors)) {
				return &audiobooks[i], "title"
			}
		}
	}

	for i, ab := range audiobooks {
		if !sharesAuthor(book.Authors, ab.Authors) {
			continue
		}
		for _, candidate := range titles(ab) {
			if looseTitlesMatch(title, candidate) {
				return &audiobooks[i], "title and author"
			}
		}
	}

	return nil, ""
}

// titles returns the edition title and, when it differs, the book title.
func titles(ab hardcover.Audiobook) []string {
	if ab.BookTitle == "" || strings.EqualFold(ab.BookTitle, ab.Title) {
		return []string{ab.Title}
	}
	return []string{ab.Title, ab.BookTitle}
}

// looseTitlesMatch reports whether two titles are the same once punctuation,
// articles and series markers are ignored, or one is the other without its subtitle.
// "Dune" matches "Dune: Book One", but "Mistborn: The Final Empire" doesn't match
// "Mistborn: The Well of Ascension".
func looseTitlesMatch(a, b string) bool {
	la, lb := looseTitle(a), looseTitle(b)
	if la == "" || lb == "" {
		return false
	}
	return la == lb || la == looseTitle(mainTitle(b)) || looseTitle(mainTitle(a)) == lb
}

// mainTitle returns a title without its subtitle.
func mainTitle(title string) string {
	if i := strings.IndexAny(title, ":–—"); i > 0 {
		title = title[:i]
	}
	if i := strings.Index(title, " - "); i > 0 {
		title = title[:i]
	}
	return title
}

// looseTitle lowercases a title, drops bracketed parts such as "(Book 1)",
// punctuation and a leading article, and collapses spaces.
func looseTitle(title string) string {
	var b strings.Builder
	depth := 0
	for _, r := range strings.ToLower(strings.ReplaceAll(title, "&", " and ")) {
		switch {
		case r == '(' || r == '[':
			depth++
		case r == ')' || r == ']':
			depth = max(depth-1, 0)
		case depth > 0:
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '\'' || r == '’':
			// "Ender's" and "Enders" are the same title.
		default:
			b.WriteRune(' ')
		}
	}
	words := strings.Fields(b.String())
	if len(words) > 1 && (words[0] == "the" || words[0] == "a" || words[0] == "an") {
		words = words[1:]
	}
	return strings.Join(words, " ")
}

// sharesAuthor reports whether any author appears in both lists, ignoring case,
// spaces and punctuation ("J.R.R. Tolkien" equals "J. R. R. Tolkien").
func sharesAuthor(a, b []string) bool {
	for _, x := range a {
		nx := normalizeName(x)
		if nx == "" {
			continue
		}
		for _, y := range b {
			if normalizeName(y) == nx {
				return true
			}
		}
	}
	return false
}

func normalizeName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, name)
}

func normalizeISBN(isbn string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(isbn)
}
