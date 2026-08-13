package security

import (
	"sync"
	"time"
)

// NonceStore records nonces and reports whether a nonce has been seen recently.
type NonceStore interface {
	// Seen returns true if nonce was already seen in the retention window.
	// It also records the nonce for future checks.
	Seen(nonce string) bool
}

// TimeBucketStore keeps nonces in per-minute buckets and retains a fixed number
// of recent buckets, giving an approximate retention window.
type TimeBucketStore struct {
	mu      sync.Mutex
	buckets map[int64]map[string]struct{}
	window  int
	nowFunc func() time.Time
}

// NewTimeBucketStore creates a store that keeps windowSize recent buckets.
func NewTimeBucketStore(windowSize int) *TimeBucketStore {
	return &TimeBucketStore{
		buckets: make(map[int64]map[string]struct{}),
		window:  windowSize,
		nowFunc: time.Now,
	}
}

// Seen reports whether nonce has been seen in the current retention window.
func (s *TimeBucketStore) Seen(nonce string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.nowFunc()
	currentBucket := now.Unix() / 60

	// Clean up buckets outside the retention window.
	for bucket := range s.buckets {
		if bucket < currentBucket-int64(s.window-1) {
			delete(s.buckets, bucket)
		}
	}

	// Search active buckets for the nonce.
	for bucket, items := range s.buckets {
		if bucket < currentBucket-int64(s.window-1) {
			continue
		}
		if _, exists := items[nonce]; exists {
			return true
		}
	}

	// Record the nonce in the current bucket.
	if s.buckets[currentBucket] == nil {
		s.buckets[currentBucket] = make(map[string]struct{})
	}
	s.buckets[currentBucket][nonce] = struct{}{}
	return false
}
