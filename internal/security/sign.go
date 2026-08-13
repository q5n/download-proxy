package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Sign generates a signed download token for the provided URL, timestamp and nonce.
func Sign(url string, timestamp int64, nonce string, secret string) string {
	data := fmt.Sprintf("url=%s&time=%d&nonce=%s", url, timestamp, nonce)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// Verify validates a signed download request against the expected HMAC signature and TTL.
func Verify(url string, timestamp int64, nonce string, sign string, secret string, maxExpire int64, store NonceStore) bool {
	now := time.Now().Unix()
	maxSkew := int64(3 * 60) // Allow a 3-minute clock skew for future timestamps.

	if timestamp < now-maxSkew {
		return false
	}
	if timestamp > now+maxSkew {
		return false
	}
	if maxExpire > 0 && now-timestamp > maxExpire {
		return false
	}
	if store != nil && store.Seen(nonce) {
		return false
	}

	expected := Sign(url, timestamp, nonce, secret)
	return hmac.Equal([]byte(expected), []byte(sign))
}
