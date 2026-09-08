package resilience

import (
	"math"
	"sync"
	"time"
)

type TokenBucket struct {
	mu              sync.Mutex
	capacity        float64
	tokens          float64
	refillPerSecond float64
	lastRefill      time.Time
}

func NewTokenBucket(capacity float64, refillPerSecond float64) *TokenBucket {
	return &TokenBucket{
		capacity:        capacity,
		tokens:          capacity, // Start full
		refillPerSecond: refillPerSecond,
		lastRefill:      time.Now(),
	}
}

func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	
	// Refill tokens based on elapsed time, capped at capacity
	b.tokens = math.Min(b.capacity, b.tokens+(elapsed*b.refillPerSecond))
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}
