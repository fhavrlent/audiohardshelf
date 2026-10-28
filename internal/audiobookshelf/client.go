// Package audiobookshelf is a minimal client for the Audiobookshelf REST API.
package audiobookshelf

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/fhavrlent/audiohardshelf/internal/httpjson"
)

const inProgressPageSize = 50

// The library item filter format the Audiobookshelf web UI uses.
var inProgressFilter = "progress." + base64.StdEncoding.EncodeToString([]byte("in-progress"))

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func New(baseURL, apiKey string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, http: &http.Client{Timeout: timeout}}
}

// MediaProgress is a user's listening progress for one library item.
type MediaProgress struct {
	LibraryItemID string  `json:"libraryItemId"`
	MediaItemType string  `json:"mediaItemType"`
	CurrentTime   float64 `json:"currentTime"`
	// Duration of the audio in seconds; zero when unknown.
	Duration   float64 `json:"duration"`
	IsFinished bool    `json:"isFinished"`
	// Epoch milliseconds; zero when unset.
	StartedAt  int64 `json:"startedAt"`
	FinishedAt int64 `json:"finishedAt"`
	LastUpdate int64 `json:"lastUpdate"`
}

// Book is the metadata needed to match a library item on Hardcover.
type Book struct {
	Title   string
	ISBN    string
	ASIN    string
	Authors []string
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	return httpjson.Do(c.http, req, out)
}

// Ping checks that the server is reachable.
func (c *Client) Ping(ctx context.Context) error {
	return c.get(ctx, "/ping", nil, nil)
}

// CurrentlyListening returns progress for books that are in progress, plus books
// finished at or after finishedSince.
//
// The library items endpoint is the source of truth for what is in progress, since
// /api/me also holds stale progress. It excludes finished books, so those come from
// /api/me.
func (c *Client) CurrentlyListening(ctx context.Context, finishedSince time.Time) ([]MediaProgress, error) {
	var libraries struct {
		Libraries []struct {
			ID        string `json:"id"`
			MediaType string `json:"mediaType"`
		} `json:"libraries"`
	}
	if err := c.get(ctx, "/api/libraries", nil, &libraries); err != nil {
		return nil, err
	}

	inProgress := map[string]bool{}
	for _, library := range libraries.Libraries {
		if library.MediaType != "book" {
			continue
		}
		ids, err := c.inProgressItemIDs(ctx, library.ID)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			inProgress[id] = true
		}
	}

	var me struct {
		MediaProgress []MediaProgress `json:"mediaProgress"`
	}
	if err := c.get(ctx, "/api/me", nil, &me); err != nil {
		return nil, err
	}

	since := finishedSince.UnixMilli()
	var result []MediaProgress
	for _, progress := range me.MediaProgress {
		if progress.MediaItemType != "book" {
			continue
		}
		if inProgress[progress.LibraryItemID] || (progress.IsFinished && progress.FinishedAt >= since) {
			result = append(result, progress)
		}
	}
	return result, nil
}

func (c *Client) inProgressItemIDs(ctx context.Context, libraryID string) ([]string, error) {
	var ids []string
	for page := 0; ; page++ {
		var items struct {
			Results []struct {
				ID string `json:"id"`
			} `json:"results"`
			Total int `json:"total"`
		}
		query := url.Values{
			"filter": {inProgressFilter},
			"sort":   {"media.metadata.title"},
			"desc":   {"0"},
			"limit":  {strconv.Itoa(inProgressPageSize)},
			"page":   {strconv.Itoa(page)},
		}
		if err := c.get(ctx, "/api/libraries/"+url.PathEscape(libraryID)+"/items", query, &items); err != nil {
			return nil, err
		}

		for _, item := range items.Results {
			if item.ID != "" {
				ids = append(ids, item.ID)
			}
		}

		if len(items.Results) < inProgressPageSize || (page+1)*inProgressPageSize >= items.Total {
			return ids, nil
		}
	}
}

// Book returns the metadata of a library item.
func (c *Client) Book(ctx context.Context, libraryItemID string) (Book, error) {
	var item struct {
		Media struct {
			Metadata struct {
				Title   string `json:"title"`
				ISBN    string `json:"isbn"`
				ASIN    string `json:"asin"`
				Authors []struct {
					Name string `json:"name"`
				} `json:"authors"`
			} `json:"metadata"`
		} `json:"media"`
	}
	if err := c.get(ctx, "/api/items/"+url.PathEscape(libraryItemID), nil, &item); err != nil {
		return Book{}, fmt.Errorf("fetching book %s: %w", libraryItemID, err)
	}
	metadata := item.Media.Metadata
	book := Book{Title: metadata.Title, ISBN: metadata.ISBN, ASIN: metadata.ASIN}
	for _, author := range metadata.Authors {
		if author.Name != "" {
			book.Authors = append(book.Authors, author.Name)
		}
	}
	return book, nil
}
