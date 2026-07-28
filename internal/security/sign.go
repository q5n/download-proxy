package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Sign generates a signed download token for the provided URL and expiration time.
func Sign(url string, expire int64, secret string) string {
	// Build the signed payload and generate the HMAC-SHA256 signature.
	data := fmt.Sprintf("url=%s&expire=%d", url, expire)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// Verify validates a signed download request against the expected HMAC signature and expiry window.
func Verify(url string, expire int64, sign string, secret string, maxExpire int64) bool {
	now := time.Now().Unix()

	// Reject expired requests immediately.
	if expire < now {
		return false
	}

	// Enforce the maximum allowed lifetime for the signed URL.
	if maxExpire > 0 && uint64(expire)-uint64(now) > uint64(maxExpire) {
		return false
	}

	expected := Sign(url, expire, secret)
	return hmac.Equal([]byte(expected), []byte(sign))
}
