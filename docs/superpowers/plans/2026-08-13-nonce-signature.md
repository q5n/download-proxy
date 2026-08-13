# nonce 随机参数签名实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在下载代理的签名验签中加入 `nonce` 随机参数，使每次签名唯一，并在服务端 5 分钟窗口内拒绝重复 nonce。

**Architecture:** 在 `internal/security` 中新增基于时间分桶的内存 nonce 缓存，修改 `Sign`/`Verify` 接口使其接收 `nonce`；在 `internal/proxy` 的 Handler 中读取并校验 `nonce`。

**Tech Stack:** Go 1.26.4+, 标准库（crypto/hmac, crypto/sha256, sync, time）

---

## 文件结构

- **新建** `internal/security/nonce.go`：`NonceStore` 接口与基于时间分桶的 `TimeBucketStore` 实现。
- **修改** `internal/security/sign.go`：`Sign` 与 `Verify` 函数签名增加 `nonce` 参数；签名串变为 `url=...&time=...&nonce=...`。
- **修改** `internal/security/sign_test.go`：更新现有测试，新增 nonce 重复与过期测试。
- **修改** `internal/proxy/proxy.go`：`Proxy` 持有 `NonceStore`，Handler 读取 `nonce` 参数并传入 `Verify`。
- **修改** `internal/proxy/proxy_test.go`：所有测试请求补充 `nonce` 参数，新增缺失 nonce 测试。
- **修改** `AGENTS.md`：更新签名格式说明。

---

### Task 1: 实现 nonce 时间分桶缓存

**Files:**
- Create: `internal/security/nonce.go`
- Test: `internal/security/nonce_test.go`

- [ ] **Step 1: 编写 `internal/security/nonce.go`**

```go
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
	mu       sync.Mutex
	buckets  map[int64]map[string]struct{}
	window   int
	nowFunc  func() time.Time
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
```

- [ ] **Step 2: 编写失败测试 `internal/security/nonce_test.go`**

```go
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
```

- [ ] **Step 3: 运行测试确认通过**

Run: `go test ./internal/security/... -run TestTimeBucketStore -v`
Expected: PASS

---

### Task 2: 更新签名/验签函数

**Files:**
- Modify: `internal/security/sign.go`
- Test: `internal/security/sign_test.go`

- [ ] **Step 1: 修改 `internal/security/sign.go`**

```go
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
```

- [ ] **Step 2: 更新 `internal/security/sign_test.go`**

```go
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
```

- [ ] **Step 3: 运行测试**

Run: `go test ./internal/security/... -v`
Expected: PASS

---

### Task 3: 更新 Proxy Handler

**Files:**
- Modify: `internal/proxy/proxy.go`
- Test: `internal/proxy/proxy_test.go`

- [ ] **Step 1: 修改 `internal/proxy/proxy.go`**

在 `Proxy` 结构体中增加 `NonceStore`：

```go
type Proxy struct {
	Config     *config.Config
	Client     *http.Client
	NonceStore security.NonceStore
}
```

在 `New` 中初始化：

```go
func New(cfg *config.Config) *Proxy {
	p := &Proxy{
		Config:     cfg,
		NonceStore: security.NewTimeBucketStore(5),
	}
	// ... existing redirect client setup ...
	return p
}
```

在 `Handler` 中读取 `nonce` 并校验：

```go
q := r.URL.Query()
target := q.Get("url")
timestampStr := q.Get("time")
nonce := q.Get("nonce")
sign := q.Get("sign")

if target == "" || timestampStr == "" || nonce == "" || sign == "" {
	log.Printf("missing request parameters target=%q time=%q nonce=%q signPresent=%t", target, timestampStr, nonce, sign != "")
	http.Error(w, "missing parameter", http.StatusBadRequest)
	return
}

// ... parse timestamp ...

if !security.Verify(target, timestamp, nonce, sign, p.Config.Secret, p.Config.MaxExpireSeconds, p.NonceStore) {
	log.Printf("signature verification failed target=%s timestamp=%d nonce=%s", target, timestamp, nonce)
	http.Error(w, "invalid signature", http.StatusForbidden)
	return
}
```

- [ ] **Step 2: 更新 `internal/proxy/proxy_test.go`**

所有构造请求的测试需要生成 `nonce` 并重新计算签名。例如：

```go
nonce := "test-nonce-1"
sign := security.Sign(target, ts, nonce, "secret")
url := fmt.Sprintf("/download?url=%s&time=%d&nonce=%s&sign=%s", url.QueryEscape(target), ts, nonce, sign)
```

新增测试：

```go
func TestHandler_MissingNonce(t *testing.T) {
	cfg := &config.Config{
		Secret:           "secret",
		MaxExpireSeconds: 3600,
		AllowedDomains:   []string{"example.com"},
	}
	p := New(cfg)

	ts := time.Now().Unix()
	target := "https://example.com/file"
	sign := security.Sign(target, ts, "any", "secret")
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/download?url=%s&time=%d&sign=%s", url.QueryEscape(target), ts, sign), nil)
	rr := httptest.NewRecorder()
	p.Handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
```

- [ ] **Step 3: 运行测试**

Run: `go test ./internal/proxy/... -v`
Expected: PASS

---

### Task 4: 更新 AGENTS.md

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: 修改签名格式段落**

将：

```
url=<target-url>&time=<unix-seconds>
```

改为：

```
url=<target-url>&time=<unix-seconds>&nonce=<nonce>
```

并在该段落后增加一行说明：`nonce` 为调用方生成的随机字符串，每次请求需唯一，服务端会在 5 分钟窗口内拒绝重复。

---

### Task 5: 全量测试与清理

- [ ] **Step 1: 运行全部测试**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 2: 运行 go vet**

Run: `go vet ./...`
Expected: no issues

---

## 自检清单

- [x] 覆盖 spec 中所有要求：nonce 参与签名、5 分钟去重、强制 nonce、Handler 读取参数。
- [x] 无 TBD/TODO/"稍后实现" 等占位符。
- [x] 类型一致性：`NonceStore` 接口、`TimeBucketStore`、函数签名在各任务中保持一致。
