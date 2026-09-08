package resilience

import (
	"sync"
	"time"
)

type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

type CircuitBreaker struct {
	mu                sync.Mutex
	state             State
	failureThreshold  int
	windowDuration    time.Duration
	openDuration      time.Duration
	failureTimestamps []time.Time
	openedAt          time.Time
}

func NewCircuitBreaker(failureThreshold int, windowDuration time.Duration, openDuration time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            Closed,
		failureThreshold: failureThreshold,
		windowDuration:   windowDuration,
		openDuration:     openDuration,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == Open {
		if time.Since(cb.openedAt) > cb.openDuration {
			cb.state = HalfOpen
			return true // exactly one probe request
		}
		return false
	}

	if cb.state == HalfOpen {
		return false // Deny further requests until probe completes (success or failure)
	}

	return true
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.failureTimestamps = append(cb.failureTimestamps, now)

	// slide the window: drop failures older than windowDuration
	cutoff := now.Add(-cb.windowDuration)
	i := 0
	for i < len(cb.failureTimestamps) && !cb.failureTimestamps[i].After(cutoff) {
		i++
	}
	cb.failureTimestamps = cb.failureTimestamps[i:]

	if cb.state == HalfOpen || len(cb.failureTimestamps) >= cb.failureThreshold {
		cb.state = Open
		cb.openedAt = now
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = Closed
	cb.failureTimestamps = nil
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
