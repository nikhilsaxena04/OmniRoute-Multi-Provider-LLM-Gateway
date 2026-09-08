package resilience

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	t.Run("Immediate consumption", func(t *testing.T) {
		bucket := NewTokenBucket(5, 1)

		// Consume all 5
		for i := 0; i < 5; i++ {
			if !bucket.Allow() {
				t.Errorf("Expected request %d to be allowed", i+1)
			}
		}

		// 6th should be rate limited
		if bucket.Allow() {
			t.Errorf("Expected 6th request to be denied")
		}
	})

	t.Run("Refill over time", func(t *testing.T) {
		bucket := NewTokenBucket(2, 1) // 2 capacity, 1 per second

		bucket.Allow()
		bucket.Allow()

		if bucket.Allow() {
			t.Errorf("Expected 3rd request to be denied initially")
		}

		// Manually rewind time to simulate passage
		bucket.mu.Lock()
		bucket.lastRefill = bucket.lastRefill.Add(-time.Second * 2)
		bucket.mu.Unlock()

		if !bucket.Allow() {
			t.Errorf("Expected request to be allowed after refill")
		}
		if !bucket.Allow() {
			t.Errorf("Expected 2nd request to be allowed after refill")
		}
		if bucket.Allow() {
			t.Errorf("Expected 3rd request to be denied after capacity is drained again")
		}
	})
}
