package resilience

import (
	"testing"
	"time"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second, time.Second*2)

	// Should start Closed
	if !cb.Allow() {
		t.Errorf("Expected cb to allow requests initially")
	}

	// Record 2 failures (under threshold of 3)
	cb.RecordFailure()
	cb.RecordFailure()

	if !cb.Allow() {
		t.Errorf("Expected cb to remain Closed under threshold")
	}

	// 3rd failure trips the breaker
	cb.RecordFailure()

	if cb.Allow() {
		t.Errorf("Expected cb to be Open after 3 failures")
	}

	// Fast forward past openDuration
	cb.mu.Lock()
	cb.openedAt = cb.openedAt.Add(-time.Second * 3)
	cb.mu.Unlock()

	// First probe allowed (HalfOpen)
	if !cb.Allow() {
		t.Errorf("Expected cb to allow probe request after openDuration")
	}

	// Second probe denied (still HalfOpen, waiting for success/fail)
	if cb.Allow() {
		t.Errorf("Expected cb to deny second request while HalfOpen")
	}

	// Probe fails -> opens again
	cb.RecordFailure()
	if cb.Allow() {
		t.Errorf("Expected cb to be Open after failed probe")
	}

	// Fast forward past openDuration again
	cb.mu.Lock()
	cb.openedAt = cb.openedAt.Add(-time.Second * 3)
	cb.mu.Unlock()

	if !cb.Allow() {
		t.Errorf("Expected cb to allow probe request")
	}

	// Probe succeeds -> closes
	cb.RecordSuccess()
	if !cb.Allow() {
		t.Errorf("Expected cb to be fully Closed and allow requests")
	}
	if !cb.Allow() {
		t.Errorf("Expected cb to be fully Closed and allow multiple requests")
	}
}

func TestCircuitBreaker_SlidingWindowCleanup(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second, time.Second*2)

	// Record 2 failures
	cb.RecordFailure()
	cb.RecordFailure()

	// Fake the timestamps so they are old
	cb.mu.Lock()
	for i := range cb.failureTimestamps {
		cb.failureTimestamps[i] = time.Now().Add(-time.Second * 2)
	}
	cb.mu.Unlock()

	// Record 1 new failure
	cb.RecordFailure()

	// The old failures should have been purged, so we only have 1 active failure
	if !cb.Allow() {
		t.Errorf("Expected cb to remain Closed since old failures should be purged")
	}
	
	cb.mu.Lock()
	if len(cb.failureTimestamps) != 1 {
		t.Errorf("Expected exactly 1 failure timestamp remaining, got %d", len(cb.failureTimestamps))
	}
	cb.mu.Unlock()
}
