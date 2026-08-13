package security

import (
	"testing"
	"time"
)

func TestSign(t *testing.T) {
	got := Sign("https://example.com/file", 1234567890, "nonce1", "secret")
	want := "03ba342442ddd66c4bf1814edd5a1b5f0c1fdef6b9da2aff0b293109b04e9aaa"
	if got != want {
		t.Errorf("Sign() = %q, want %q", got, want)
	}
}

func TestVerify(t *testing.T) {
	url := "https://example.com/file"
	secret := "secret"
	maxExpire := int64(3600)
	store := NewTimeBucketStore(5)

	t.Run("valid signature within ttl", func(t *testing.T) {
		ts := time.Now().Unix()
		sign := Sign(url, ts, "nonce-a", secret)
		if !Verify(url, ts, "nonce-a", sign, secret, maxExpire, store) {
			t.Error("Verify() returned false for a valid signature")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		ts := int64(1234567890)
		sign := Sign(url, ts, "nonce-b", secret)
		if Verify(url, ts, "nonce-b", sign+"bad", secret, maxExpire, store) {
			t.Error("Verify() returned true for an invalid signature")
		}
	})

	t.Run("expired timestamp", func(t *testing.T) {
		ts := int64(0)
		sign := Sign(url, ts, "nonce-c", secret)
		if Verify(url, ts, "nonce-c", sign, secret, maxExpire, store) {
			t.Error("Verify() returned true for an expired timestamp")
		}
	})

	t.Run("future timestamp beyond tolerance", func(t *testing.T) {
		ts := time.Now().Add(6 * time.Minute).Unix()
		sign := Sign(url, ts, "nonce-d", secret)
		if Verify(url, ts, "nonce-d", sign, secret, maxExpire, store) {
			t.Error("Verify() returned true for a timestamp more than 5 minutes in the future")
		}
	})

	t.Run("future timestamp within tolerance", func(t *testing.T) {
		ts := time.Now().Add(2 * time.Minute).Unix()
		sign := Sign(url, ts, "nonce-e", secret)
		if !Verify(url, ts, "nonce-e", sign, secret, maxExpire, store) {
			t.Error("Verify() returned false for a timestamp within 5 minutes in the future")
		}
	})

	t.Run("reused nonce", func(t *testing.T) {
		ts := time.Now().Unix()
		sign := Sign(url, ts, "nonce-reuse", secret)
		if !Verify(url, ts, "nonce-reuse", sign, secret, maxExpire, store) {
			t.Error("Verify() returned false for first use of nonce")
		}
		if Verify(url, ts, "nonce-reuse", sign, secret, maxExpire, store) {
			t.Error("Verify() returned true for reused nonce")
		}
	})
}
