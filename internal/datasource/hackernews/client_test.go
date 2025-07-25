package hackernews

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetMaxItemID(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/maxitem.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("35000000"))
	}))
	defer server.Close()

	// Create client with test server
	client := NewClient()
	client.httpClient = server.Client()
	client.baseURL = server.URL // Use test server URL

	ctx := context.Background()

	// Test successful request
	maxID, err := client.GetMaxItemID(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(35000000), maxID)
}

func TestClient_GetItem(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/item/")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 1,
			"type": "story",
			"by": "testuser",
			"time": 1160418111,
			"title": "Test Story",
			"url": "http://example.com",
			"score": 100,
			"descendants": 5,
			"kids": [2, 3, 4]
		}`))
	}))
	defer server.Close()

	client := NewClient()
	client.httpClient = server.Client()
	client.baseURL = server.URL

	ctx := context.Background()

	// Test successful request
	item, err := client.GetItem(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, item)

	assert.Equal(t, int64(1), item.ID)
	assert.Equal(t, "story", item.Type)
	assert.Equal(t, "testuser", item.By)
	assert.Equal(t, int64(1160418111), item.Time)
	assert.Equal(t, "Test Story", item.Title)
	assert.Equal(t, "http://example.com", item.URL)
	assert.Equal(t, int64(100), item.Score)
	assert.Equal(t, int64(5), item.Descendants)
	assert.Equal(t, []int64{2, 3, 4}, item.Kids)
}

func TestClient_GetItem_Null(t *testing.T) {
	// Mock server returning null (deleted item)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("null"))
	}))
	defer server.Close()

	client := NewClient()
	client.httpClient = server.Client()
	client.baseURL = server.URL

	ctx := context.Background()

	// Test null item (should return nil, nil)
	item, err := client.GetItem(ctx, 999)
	require.NoError(t, err)
	assert.Nil(t, item)
}

func TestClient_GetItemsBatch(t *testing.T) {
	callCount := 0
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Return different items based on call count
		if callCount == 1 {
			w.Write([]byte(`{"id": 1, "type": "story", "title": "Item 1"}`))
		} else if callCount == 2 {
			w.Write([]byte("null")) // Deleted item
		} else {
			w.Write([]byte(`{"id": 3, "type": "comment", "text": "Item 3"}`))
		}
	}))
	defer server.Close()

	client := NewClient()
	client.httpClient = server.Client()
	client.baseURL = server.URL

	ctx := context.Background()

	// Test batch request
	items, err := client.GetItemsBatch(ctx, 1, 3)
	require.NoError(t, err)

	// Should get 2 items (1 and 3), item 2 is null/deleted
	assert.Len(t, items, 2)
	assert.Equal(t, int64(1), items[0].ID)
	assert.Equal(t, "story", items[0].Type)
	assert.Equal(t, int64(3), items[1].ID)
	assert.Equal(t, "comment", items[1].Type)
}

func TestClient_GetItemsBatch_InvalidRange(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test invalid range
	_, err := client.GetItemsBatch(ctx, 10, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "startID")
}

func TestRateLimiter(t *testing.T) {
	// Test rate limiter with very low rate
	rl := NewRateLimiter(2, time.Second)
	defer rl.Close()

	ctx := context.Background()
	start := time.Now()

	// First two calls should be immediate
	err := rl.Wait(ctx)
	assert.NoError(t, err)

	err = rl.Wait(ctx)
	assert.NoError(t, err)

	// Third call should be delayed
	err = rl.Wait(ctx)
	assert.NoError(t, err)

	elapsed := time.Since(start)
	// Should take at least 400ms (2 tokens per second = 500ms per token, but with some tolerance)
	assert.True(t, elapsed >= 400*time.Millisecond, "Rate limiting not working: elapsed %v", elapsed)
}

func TestRateLimiter_Context_Cancellation(t *testing.T) {
	rl := NewRateLimiter(1, time.Second)
	defer rl.Close()

	// Use up the token
	ctx := context.Background()
	err := rl.Wait(ctx)
	assert.NoError(t, err)

	// Create cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should return immediately with context error
	err = rl.Wait(cancelledCtx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
