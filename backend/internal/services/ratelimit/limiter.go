package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
}

func New() *Limiter {
	return &Limiter{buckets: map[string][]time.Time{}}
}

func (l *Limiter) Allow(key string, limit int) bool {
	if limit <= 0 {
		return true
	}
	now := time.Now()
	windowStart := now.Add(-time.Minute)
	l.mu.Lock()
	defer l.mu.Unlock()

	events := l.buckets[key]
	kept := events[:0]
	for _, event := range events {
		if event.After(windowStart) {
			kept = append(kept, event)
		}
	}
	if len(kept) >= limit {
		l.buckets[key] = kept
		return false
	}
	l.buckets[key] = append(kept, now)
	return true
}
