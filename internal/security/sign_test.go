package security

import (
	"testing"
	"time"
)

func TestSign(t *testing.T) {
	got := Sign("https://example.com/file", 1234567890, "secret")
	want := "a3e96a32d6db8907f95896b209b9c9d18bebab67246fb2f2bb1701ba9c4e68a2"
	if got != want {
		t.Errorf("Sign() = %q, want %q", got, want)
	}
}

func TestVerify(t *testing.T) {
	url := "https://example.com/file"
	secret := "secret"
	maxExpire := int64(3600)

	t.Run("valid signature within ttl", func(t *testing.T) {
		ts := time.Now().Unix()
		sign := Sign(url, ts, secret)
		if !Verify(url, ts, sign, secret, maxExpire) {
			t.Error("Verify() returned false for a valid signature")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		ts := int64(1234567890)
		sign := Sign(url, ts, secret)
		if Verify(url, ts, sign+"bad", secret, maxExpire) {
			t.Error("Verify() returned true for an invalid signature")
		}
	})

	t.Run("expired timestamp", func(t *testing.T) {
		ts := int64(0)
		sign := Sign(url, ts, secret)
		if Verify(url, ts, sign, secret, maxExpire) {
			t.Error("Verify() returned true for an expired timestamp")
		}
	})

	t.Run("future timestamp beyond tolerance", func(t *testing.T) {
		ts := time.Now().Add(6 * time.Minute).Unix()
		sign := Sign(url, ts, secret)
		if Verify(url, ts, sign, secret, maxExpire) {
			t.Error("Verify() returned true for a timestamp more than 5 minutes in the future")
		}
	})

	t.Run("future timestamp within tolerance", func(t *testing.T) {
		ts := time.Now().Add(2 * time.Minute).Unix()
		sign := Sign(url, ts, secret)
		if !Verify(url, ts, sign, secret, maxExpire) {
			t.Error("Verify() returned false for a timestamp within 5 minutes in the future")
		}
	})
}
