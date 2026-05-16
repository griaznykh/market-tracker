package ratelimiter

import (
	"context"
	"math"
	"sync"
	"time"
)

type TokenBucket struct {
	mu             sync.Mutex
	tokens         float64
	maxTokens      float64
	refillRate     float64
	lastRefillTime time.Time
	lastSeen       time.Time
}

func NewTokenBucket(maxTokens, refillRate float64) *TokenBucket {
	now := time.Now()
	return &TokenBucket{
		tokens:         maxTokens,
		maxTokens:      maxTokens,
		refillRate:     refillRate,
		lastRefillTime: now,
		lastSeen:       now,
	}
}

func (tb *TokenBucket) refill(now time.Time) {
	duration := now.Sub(tb.lastRefillTime)
	tb.tokens = math.Min(tb.tokens+tb.refillRate*duration.Seconds(), tb.maxTokens)
	tb.lastRefillTime = now
}

func (tb *TokenBucket) Allow(n float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	tb.refill(now)
	tb.lastSeen = now

	if tb.tokens >= n {
		tb.tokens -= n

		return true
	}
	return false
}

type RateLimiterManager struct {
	mu      sync.Mutex
	buckets map[string]*TokenBucket
	ttl     time.Duration
	limit   float64
}

func NewRateLimiterManager(ctx context.Context, limit float64, ttl, cleanupInterval time.Duration) *RateLimiterManager {
	m := &RateLimiterManager{
		buckets: make(map[string]*TokenBucket),
		limit:   limit,
		ttl:     ttl,
	}

	go m.cleanupWorker(ctx, cleanupInterval)

	return m
}

func (m *RateLimiterManager) getBucket(id string) *TokenBucket {
	m.mu.Lock()
	defer m.mu.Unlock()

	if b, ok := m.buckets[id]; ok {
		return b
	}

	b := NewTokenBucket(m.limit, m.limit/60)
	m.buckets[id] = b
	return b
}

func (m *RateLimiterManager) Request(id string) bool {
	return m.getBucket(id).Allow(1)
}

func (m *RateLimiterManager) cleanupWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

func (m *RateLimiterManager) cleanup() {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	for id, bucket := range m.buckets {
		bucket.mu.Lock()
		lastSeen := bucket.lastSeen
		bucket.mu.Unlock()

		if now.Sub(lastSeen) > m.ttl {
			delete(m.buckets, id)
		}
	}
}
