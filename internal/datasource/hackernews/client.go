package hackernews

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var (
	BaseURL        = "https://hacker-news.firebaseio.com/v0"
	MaxItemURL     = BaseURL + "/maxitem.json"
	ItemURL        = BaseURL + "/item/%d.json"
	DefaultTimeout = 30 * time.Second
)

// Client represents a Hacker News API client
type Client struct {
	httpClient  *http.Client
	rateLimiter *RateLimiter
	baseURL     string
}

// Item represents a Hacker News item
type Item struct {
	ID          int64   `json:"id"`
	Type        string  `json:"type"`
	By          string  `json:"by"`
	Time        int64   `json:"time"`
	Text        string  `json:"text"`
	Dead        bool    `json:"dead"`
	Deleted     bool    `json:"deleted"`
	Parent      int64   `json:"parent"`
	Kids        []int64 `json:"kids"`
	URL         string  `json:"url"`
	Score       int64   `json:"score"`
	Title       string  `json:"title"`
	Descendants int64   `json:"descendants"`
}

// NewClient creates a new Hacker News API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		rateLimiter: NewRateLimiter(10, time.Second), // 10 requests per second
		baseURL:     BaseURL,
	}
}

// GetMaxItemID fetches the current maximum item ID from the API
func (c *Client) GetMaxItemID(ctx context.Context) (int64, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return 0, fmt.Errorf("rate limiter error: %w", err)
	}

	url := c.baseURL + "/maxitem.json"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch max item ID: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var maxID int64
	if err := json.NewDecoder(resp.Body).Decode(&maxID); err != nil {
		return 0, fmt.Errorf("failed to decode max item ID: %w", err)
	}

	return maxID, nil
}

// GetItem fetches a specific item by ID from the API
func (c *Client) GetItem(ctx context.Context, id int64) (*Item, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	url := fmt.Sprintf(c.baseURL+"/item/%d.json", id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch item %d: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d for item %d", resp.StatusCode, id)
	}

	var item Item
	// Check if response is null first
	var rawJson json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&rawJson); err != nil {
		return nil, fmt.Errorf("failed to decode response for item %d: %w", id, err)
	}

	// If response is null, item doesn't exist or is deleted
	if string(rawJson) == "null" {
		return nil, nil
	}

	// Decode the actual item
	if err := json.Unmarshal(rawJson, &item); err != nil {
		return nil, fmt.Errorf("failed to decode item %d: %w", id, err)
	}

	return &item, nil
}

// GetItemsBatch fetches multiple items in a batch
func (c *Client) GetItemsBatch(ctx context.Context, startID, endID int64) ([]*Item, error) {
	if startID > endID {
		return nil, fmt.Errorf("startID (%d) must be <= endID (%d)", startID, endID)
	}

	items := make([]*Item, 0, endID-startID+1)

	for id := startID; id <= endID; id++ {
		select {
		case <-ctx.Done():
			return items, ctx.Err()
		default:
		}

		item, err := c.GetItem(ctx, id)
		if err != nil {
			return items, fmt.Errorf("failed to get item %d: %w", id, err)
		}

		// Item can be nil if it doesn't exist or is deleted
		if item != nil {
			items = append(items, item)
		}
	}

	return items, nil
}
