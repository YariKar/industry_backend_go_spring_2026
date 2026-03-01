package main

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type Limiter struct {
	mu       sync.Mutex
	clock    Clock
	rate     float64
	burst    int
	tokens   float64
	lastTime time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	return &Limiter{
		clock:    clock,
		rate:     ratePerSec,
		burst:    burst,
		tokens:   float64(burst),
		lastTime: clock.Now(),
	}
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()
	if now.After(l.lastTime) {
		elapsed := now.Sub(l.lastTime).Seconds()
		newTokens := l.tokens + elapsed*l.rate
		if newTokens > float64(l.burst) {
			newTokens = float64(l.burst)
		}
		l.tokens = newTokens
		l.lastTime = now
	}

	if l.tokens >= 1.0 {
		l.tokens -= 1.0
		return true
	}
	return false
}