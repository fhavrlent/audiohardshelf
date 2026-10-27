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
	Title      string
	ISBN10     string
	ISBN13     string
	ASIN       string
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
        editions(where: {audio_seconds: {_gte: 0}}) {
          id
          isbn_10
          isbn_13
          asin
          title
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
				Title    string `json:"title"`
				Editions []struct {
					ID     int    `json:"id"`
					ISBN10 string `json:"isbn_10"`
					ISBN13 string `json:"isbn_13"`
					ASIN   string `json:"asin"`
					Title  string `json:"title"`
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
		for _, edition := range userBook.Book.Editions {
			if edition.ID != *userBook.EditionID {
				continue
			}
			title := edition.Title
			if title == "" {
				title = userBook.Book.Title
			}
			audiobooks = append(audiobooks, Audiobook{
				UserBookID: userBook.ID,
				EditionID:  edition.ID,
				Title:      title,
				ISBN10:     edition.ISBN10,
				ISBN13:     edition.ISBN13,
				ASIN:       edition.ASIN,
			})
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
	return c.mutate(ctx, updateReadMutation, map[string]any{"id": readID, "object": update})
}

const updateUserBookMutation = `
  mutation UpdateUserBook($id: Int!, $object: UserBookUpdateInput!) {
    result: update_user_book(id: $id, object: $object) {
      error
    }
  }`

// MarkAsRead sets a user book's status to Read.
func (c *Client) MarkAsRead(ctx context.Context, userBookID, editionID int) error {
	object := map[string]any{"edition_id": editionID, "status_id": StatusRead}
	return c.mutate(ctx, updateUserBookMutation, map[string]any{"id": userBookID, "object": object})
}

func (c *Client) mutate(ctx context.Context, mutation string, variables map[string]any) error {
	var data struct {
		Result struct {
			Error *string `json:"error"`
		} `json:"result"`
	}
	if err := c.query(ctx, mutation, variables, &data); err != nil {
		return err
	}
	if data.Result.Error != nil && *data.Result.Error != "" {
		return errors.New(*data.Result.Error)
	}
	return nil
}
