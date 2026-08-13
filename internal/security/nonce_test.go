package security

import (
	"testing"
	"time"
)

func TestTimeBucketStore_Seen(t *testing.T) {
	fixed := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	store := NewTimeBucketStore(5)
	store.nowFunc = func() time.Time { return fixed }

	if store.Seen("abc") {
		t.Error("first Seen(abc) should return false")
	}
	if !store.Seen("abc") {
		t.Error("second Seen(abc) should return true")
	}
	if store.Seen("def") {
		t.Error("first Seen(def) should return false")
	}
}

func TestTimeBucketStore_WindowExpiration(t *testing.T) {
	fixed := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	store := NewTimeBucketStore(5)
	store.nowFunc = func() time.Time { return fixed }

	store.Seen("old")

	// Advance by 5 minutes: the bucket containing "old" should be evicted.
	store.nowFunc = func() time.Time { return fixed.Add(5 * time.Minute) }
	if store.Seen("old") {
		t.Error("nonce should be evicted after window expires")
	}
}
