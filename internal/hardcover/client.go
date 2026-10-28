// Package hardcover is a minimal client for the Hardcover GraphQL API.
package hardcover

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fhavrlent/audiohardshelf/internal/httpjson"
)

// Hardcover `user_books.status_id` values.
const (
	StatusWantToRead       = 1
	StatusCurrentlyReading = 2
	StatusRead             = 3
)

type Client struct {
	url           string
	authorization string
	http          *http.Client
}

// New creates a client. apiKey may include the "Bearer " prefix or not.
func New(apiURL, apiKey string, timeout time.Duration) *Client {
	if !strings.HasPrefix(apiKey, "Bearer ") {
		apiKey = "Bearer " + apiKey
	}
	return &Client{url: apiURL, authorization: apiKey, http: &http.Client{Timeout: timeout}}
}

type User struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	BooksCount int    `json:"books_count"`
}

// Audiobook is an audio edition the user is currently reading.
type Audiobook struct {
	UserBookID int
	EditionID  int
	// Title is the edition's title, or the book's when the edition has none.
	Title     string
	BookTitle string
	ISBN10    string
	ISBN13    string
	ASIN      string
	// AudioSeconds is the edition's length; zero when unknown.
	AudioSeconds int
	Authors      []string
}

// Edition is an audio edition in the Hardcover catalog.
type Edition struct {
	ID           int    `json:"id"`
	BookID       int    `json:"book_id"`
	Title        string `json:"title"`
	AudioSeconds int    `json:"audio_seconds"`
}

// UserBook is a book on the user's shelf.
type UserBook struct {
	ID        int  `json:"id"`
	StatusID  int  `json:"status_id"`
	EditionID *int `json:"edition_id"`
}

// Read is one read-through of a user book.
type Read struct {
	ID              int    `json:"id"`
	ProgressSeconds int    `json:"progress_seconds"`
	StartedAt       string `json:"started_at"`
	FinishedAt      string `json:"finished_at"`
}

// ReadUpdate holds the fields written to a read. Dates are YYYY-MM-DD; empty dates are left unchanged.
type ReadUpdate struct {
	EditionID       int    `json:"edition_id"`
	ProgressPages   int    `json:"progress_pages"`
	ProgressSeconds int    `json:"progress_seconds"`
	StartedAt       string `json:"started_at,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
}

func (c *Client) query(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authorization)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AudioHardShelf - Audiobook Sync Tool")

	var resp struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := httpjson.Do(c.http, req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		messages := make([]string, len(resp.Errors))
		for i, e := range resp.Errors {
			messages[i] = e.Message
		}
		return fmt.Errorf("graphql: %s", strings.Join(messages, "; "))
	}
	return json.Unmarshal(resp.Data, out)
}

const meQuery = `
  query {
    me {
      id
      username
      books_count
    }
  }`

// Me returns the user the API key belongs to.
func (c *Client) Me(ctx context.Context) (User, error) {
	var data struct {
		Me json.RawMessage `json:"me"`
	}
	if err := c.query(ctx, meQuery, nil, &data); err != nil {
		return User{}, err
	}

	// Hardcover returns `me` as a single-element list; accept a plain object too.
	var user User
	var users []User
	if err := json.Unmarshal(data.Me, &users); err == nil {
		if len(users) > 0 {
			user = users[0]
		}
	} else if err := json.Unmarshal(data.Me, &user); err != nil {
		return User{}, fmt.Errorf("decoding user: %w", err)
	}
	if user.ID == 0 {
		return User{}, errors.New("hardcover returned no user for this API key")
	}
	return user, nil
}

const currentlyReadingQuery = `
  query CurrentlyReading($userId: Int!, $statusId: Int!) {
    user_books(where: {status_id: {_eq: $statusId}, user_id: {_eq: $userId}}) {
      id
      edition_id
      book {
        title
        contributions {
          author {
            name
          }
        }
        editions(where: {audio_seconds: {_gte: 0}}) {
          id
          isbn_10
          isbn_13
          asin
          title
          audio_seconds
        }
      }
    }
  }`

// CurrentlyReadingAudiobooks returns the books the user is currently reading
// in an audio edition. Books read in another edition are left out.
func (c *Client) CurrentlyReadingAudiobooks(ctx context.Context, userID int) ([]Audiobook, error) {
	var data struct {
		UserBooks []struct {
			ID        int  `json:"id"`
			EditionID *int `json:"edition_id"`
			Book      struct {
				Title         string `json:"title"`
				Contributions []struct {
					Author *struct {
						Name string `json:"name"`
					} `json:"author"`
				} `json:"contributions"`
				Editions []struct {
					ID           int    `json:"id"`
					ISBN10       string `json:"isbn_10"`
					ISBN13       string `json:"isbn_13"`
					ASIN         string `json:"asin"`
					Title        string `json:"title"`
					AudioSeconds *int   `json:"audio_seconds"`
				} `json:"editions"`
			} `json:"book"`
		} `json:"user_books"`
	}
	variables := map[string]any{"userId": userID, "statusId": StatusCurrentlyReading}
	if err := c.query(ctx, currentlyReadingQuery, variables, &data); err != nil {
		return nil, err
	}

	var audiobooks []Audiobook
	for _, userBook := range data.UserBooks {
		if userBook.EditionID == nil {
			continue
		}
		var authors []string
		for _, contribution := range userBook.Book.Contributions {
			if contribution.Author != nil && contribution.Author.Name != "" {
				authors = append(authors, contribution.Author.Name)
			}
		}
		for _, edition := range userBook.Book.Editions {
			if edition.ID != *userBook.EditionID {
				continue
			}
			title := edition.Title
			if title == "" {
				title = userBook.Book.Title
			}
			audiobook := Audiobook{
				UserBookID: userBook.ID,
				EditionID:  edition.ID,
				Title:      title,
				BookTitle:  userBook.Book.Title,
				ISBN10:     edition.ISBN10,
				ISBN13:     edition.ISBN13,
				ASIN:       edition.ASIN,
				Authors:    authors,
			}
			if edition.AudioSeconds != nil {
				audiobook.AudioSeconds = *edition.AudioSeconds
			}
			audiobooks = append(audiobooks, audiobook)
		}
	}
	return audiobooks, nil
}

const latestReadQuery = `
  query LatestRead($userBookId: Int!) {
    user_book_reads(
      where: {user_book_id: {_eq: $userBookId}}
      order_by: {id: desc}
      limit: 1
    ) {
      id
      started_at
      finished_at
      progress_seconds
    }
  }`

// LatestRead returns the most recent read of a user book (re-reads add reads),
// or nil if there is none.
func (c *Client) LatestRead(ctx context.Context, userBookID int) (*Read, error) {
	var data struct {
		Reads []Read `json:"user_book_reads"`
	}
	if err := c.query(ctx, latestReadQuery, map[string]any{"userBookId": userBookID}, &data); err != nil {
		return nil, err
	}
	if len(data.Reads) == 0 {
		return nil, nil
	}
	return &data.Reads[0], nil
}

const updateReadMutation = `
  mutation UpdateRead($id: Int!, $object: DatesReadInput!) {
    result: update_user_book_read(id: $id, object: $object) {
      error
    }
  }`

// UpdateRead writes progress and dates to a read.
func (c *Client) UpdateRead(ctx context.Context, readID int, update ReadUpdate) error {
	_, err := c.mutate(ctx, updateReadMutation, map[string]any{"id": readID, "object": update})
	return err
}

const insertReadMutation = `
  mutation InsertRead($userBookId: Int!, $read: DatesReadInput!) {
    result: insert_user_book_read(user_book_id: $userBookId, user_book_read: $read) {
      id
      error
    }
  }`

// InsertRead adds a read to a user book and returns its ID.
func (c *Client) InsertRead(ctx context.Context, userBookID int, read ReadUpdate) (int, error) {
	return c.mutate(ctx, insertReadMutation, map[string]any{"userBookId": userBookID, "read": read})
}

const updateUserBookMutation = `
  mutation UpdateUserBook($id: Int!, $object: UserBookUpdateInput!) {
    result: update_user_book(id: $id, object: $object) {
      error
    }
  }`

// MarkAsRead sets a user book's status to Read.
func (c *Client) MarkAsRead(ctx context.Context, userBookID, editionID int) error {
	return c.setStatus(ctx, userBookID, editionID, StatusRead)
}

// StartReading sets a user book's status to Currently Reading in the given edition.
func (c *Client) StartReading(ctx context.Context, userBookID, editionID int) error {
	return c.setStatus(ctx, userBookID, editionID, StatusCurrentlyReading)
}

func (c *Client) setStatus(ctx context.Context, userBookID, editionID, statusID int) error {
	object := map[string]any{"edition_id": editionID, "status_id": statusID}
	_, err := c.mutate(ctx, updateUserBookMutation, map[string]any{"id": userBookID, "object": object})
	return err
}

const insertUserBookMutation = `
  mutation InsertUserBook($object: UserBookCreateInput!) {
    result: insert_user_book(object: $object) {
      id
      error
    }
  }`

// AddCurrentlyReading puts a book on the user's shelf as Currently Reading in
// the given edition and returns the new user book's ID.
func (c *Client) AddCurrentlyReading(ctx context.Context, bookID, editionID int) (int, error) {
	object := map[string]any{"book_id": bookID, "edition_id": editionID, "status_id": StatusCurrentlyReading}
	return c.mutate(ctx, insertUserBookMutation, map[string]any{"object": object})
}

const audioEditionQuery = `
  query AudioEdition($where: editions_bool_exp!) {
    editions(where: $where, order_by: {users_count: desc}, limit: 1) {
      id
      book_id
      title
      audio_seconds
    }
  }`

// FindAudioEdition returns the audio edition with the given ISBN (10 or 13
// digits, without hyphens) or ASIN, trying ISBN first, or nil if there is none.
// Empty identifiers are skipped.
func (c *Client) FindAudioEdition(ctx context.Context, isbn, asin string) (*Edition, error) {
	var conditions []map[string]any
	switch len(isbn) {
	case 10:
		conditions = append(conditions, map[string]any{"isbn_10": map[string]any{"_eq": isbn}})
	case 13:
		conditions = append(conditions, map[string]any{"isbn_13": map[string]any{"_eq": isbn}})
	}
	if asin != "" {
		conditions = append(conditions, map[string]any{"asin": map[string]any{"_eq": asin}})
	}

	for _, condition := range conditions {
		condition["audio_seconds"] = map[string]any{"_gte": 0}
		var data struct {
			Editions []Edition `json:"editions"`
		}
		if err := c.query(ctx, audioEditionQuery, map[string]any{"where": condition}, &data); err != nil {
			return nil, err
		}
		if len(data.Editions) > 0 {
			return &data.Editions[0], nil
		}
	}
	return nil, nil
}

const userBookQuery = `
  query UserBook($userId: Int!, $bookId: Int!) {
    user_books(where: {user_id: {_eq: $userId}, book_id: {_eq: $bookId}}, limit: 1) {
      id
      status_id
      edition_id
    }
  }`

// UserBook returns the user's shelf entry for a book, or nil if it isn't on their shelf.
func (c *Client) UserBook(ctx context.Context, userID, bookID int) (*UserBook, error) {
	var data struct {
		UserBooks []UserBook `json:"user_books"`
	}
	if err := c.query(ctx, userBookQuery, map[string]any{"userId": userID, "bookId": bookID}, &data); err != nil {
		return nil, err
	}
	if len(data.UserBooks) == 0 {
		return nil, nil
	}
	return &data.UserBooks[0], nil
}

// mutate runs a mutation aliased as `result` and returns the ID it reports, if any.
func (c *Client) mutate(ctx context.Context, mutation string, variables map[string]any) (int, error) {
	var data struct {
		Result struct {
			ID    *int    `json:"id"`
			Error *string `json:"error"`
		} `json:"result"`
	}
	if err := c.query(ctx, mutation, variables, &data); err != nil {
		return 0, err
	}
	if data.Result.Error != nil && *data.Result.Error != "" {
		return 0, errors.New(*data.Result.Error)
	}
	if data.Result.ID == nil {
		return 0, nil
	}
	return *data.Result.ID, nil
}
