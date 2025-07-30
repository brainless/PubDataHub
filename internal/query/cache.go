package query

import (
	"sync"
	"time"
)

// InMemoryQueryCache implements a simple in-memory cache for query results
type InMemoryQueryCache struct {
	mu        sync.RWMutex
	items     map[string]*cacheItem
	maxSize   int
	hitCount  int64
	missCount int64
}

// cacheItem represents a cached query result with TTL
type cacheItem struct {
	result QueryResult
	expiry time.Time
}

// NewInMemoryQueryCache creates a new in-memory query cache
func NewInMemoryQueryCache(maxSize int) *InMemoryQueryCache {
	cache := &InMemoryQueryCache{
		items:   make(map[string]*cacheItem),
		maxSize: maxSize,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached query result
func (c *InMemoryQueryCache) Get(key string) (QueryResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		c.missCount++
		return QueryResult{}, false
	}

	// Check if expired
	if time.Now().After(item.expiry) {
		c.missCount++
		// Remove expired item (will be cleaned up by cleanup goroutine)
		return QueryResult{}, false
	}

	c.hitCount++
	return item.result, true
}

// Set stores a query result in the cache with TTL
func (c *InMemoryQueryCache) Set(key string, result QueryResult, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict items
	if len(c.items) >= c.maxSize {
		c.evictLRU()
	}

	c.items[key] = &cacheItem{
		result: result,
		expiry: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a specific key from the cache
func (c *InMemoryQueryCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

// Clear removes all items from the cache
func (c *InMemoryQueryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	c.hitCount = 0
	c.missCount = 0
	return nil
}

// Stats returns cache statistics
func (c *InMemoryQueryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalRequests := c.hitCount + c.missCount
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(c.hitCount) / float64(totalRequests)
	}

	// Calculate approximate memory usage
	memoryUsage := int64(len(c.items) * 1024) // Rough estimate

	return CacheStats{
		HitCount:    c.hitCount,
		MissCount:   c.missCount,
		HitRate:     hitRate,
		Size:        len(c.items),
		MaxSize:     c.maxSize,
		MemoryUsage: memoryUsage,
	}
}

// evictLRU removes the least recently used item (simplified implementation)
func (c *InMemoryQueryCache) evictLRU() {
	// Simple implementation: remove first item found
	// In a real implementation, you'd want to track access times
	for key := range c.items {
		delete(c.items, key)
		break
	}
}

// cleanup removes expired items periodically
func (c *InMemoryQueryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiry) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
