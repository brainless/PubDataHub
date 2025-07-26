package hackernews

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens   chan struct{}
	ticker   *time.Ticker
	rate     int
	interval time.Duration
	mu       sync.Mutex
	closed   bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:   make(chan struct{}, rate),
		rate:     rate,
		interval: interval,
	}

	// Fill initial tokens
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}

	// Start token refill goroutine
	rl.ticker = time.NewTicker(interval / time.Duration(rate))
	go rl.refillTokens()

	return rl
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	if rl.closed {
		rl.mu.Unlock()
		return context.Canceled
	}
	rl.mu.Unlock()

	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops the rate limiter
func (rl *RateLimiter) Close() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.closed {
		rl.closed = true
		if rl.ticker != nil {
			rl.ticker.Stop()
		}
		close(rl.tokens)
	}
}

// refillTokens adds tokens to the bucket at the specified rate
func (rl *RateLimiter) refillTokens() {
	for range rl.ticker.C {
		rl.mu.Lock()
		if rl.closed {
			rl.mu.Unlock()
			return
		}
		rl.mu.Unlock()

		// Try to add a token, but don't block if bucket is full
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}
}
