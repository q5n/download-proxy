# Replace `expire` with `time` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change the signed download URL parameter from `expire` (absolute expiration timestamp) to `time` (URL generation timestamp) and let the server enforce the TTL using `max_expire_seconds`.

**Architecture:** Update the HMAC signing payload and validation in `internal/security`, switch the HTTP handler to read `time`, update the existing end-to-end tests, and synchronize README and helper scripts.

**Tech Stack:** Go 1.26.4, standard library only for implementation; `gopkg.in/yaml.v3` for config.

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/security/sign.go` | HMAC signing/verification payload and elapsed-time check |
| `internal/security/sign_test.go` | Unit tests for signing and verification |
| `internal/proxy/proxy.go` | HTTP handler query-parameter parsing |
| `internal/proxy/proxy_test.go` | End-to-end handler tests |
| `README.md` | User-facing documentation and example |
| `scripts/test-download.sh` | Bash helper that constructs test URLs |
| `scripts/test-download.ps1` | PowerShell helper that constructs test URLs |

---

## Task 1: Update `internal/security` to use `time`

**Files:**
- Create: `internal/security/sign_test.go`
- Modify: `internal/security/sign.go`

- [ ] **Step 1: Write the failing unit test**

Create `internal/security/sign_test.go`:

```go
package security

import "testing"

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
		ts := int64(1234567890)
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
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
go test ./internal/security/... -v
```

Expected: FAIL because `Sign` still uses `expire` in the payload.

- [ ] **Step 3: Update `internal/security/sign.go`**

Replace the file contents with:

```go
package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Sign generates a signed download token for the provided URL and generation timestamp.
func Sign(url string, timestamp int64, secret string) string {
	data := fmt.Sprintf("url=%s&time=%d", url, timestamp)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// Verify validates a signed download request against the expected HMAC signature and TTL.
func Verify(url string, timestamp int64, sign string, secret string, maxExpire int64) bool {
	now := time.Now().Unix()

	if maxExpire > 0 && now-timestamp > maxExpire {
		return false
	}

	expected := Sign(url, timestamp, secret)
	return hmac.Equal([]byte(expected), []byte(sign))
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
go test ./internal/security/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/security/sign.go internal/security/sign_test.go
git commit -m "refactor(security): replace expire with time in HMAC signing"
```

---

## Task 2: Update the proxy handler and end-to-end tests

**Files:**
- Modify: `internal/proxy/proxy.go`
- Modify: `internal/proxy/proxy_test.go`

- [ ] **Step 1: Update the end-to-end tests**

Modify `internal/proxy/proxy_test.go`:

1. Replace the helper functions with:

```go
func signURL(target string, timestamp int64) string {
	return security.Sign(target, timestamp, testSecret)
}

func buildProxyURL(proxy, target string, timestamp int64) string {
	return fmt.Sprintf("%s/download?url=%s&time=%d&sign=%s",
		proxy, url.QueryEscape(target), timestamp, signURL(target, timestamp))
}
```

2. Update the test cases:
   - Keep `normal download`, `redirect to allowed host`, `redirect to disallowed host`, `invalid signature`, `blocked domain`, `missing parameter`, and `disallowed method`.
   - Replace the `expired timestamp` case with:
     ```go
     {
         name: "expired timestamp",
         makeURL: func() string {
             timestamp := time.Now().Add(-2 * time.Hour).Unix()
             return buildProxyURL(proxy.URL, upstream.URL+"/file", timestamp)
         },
         method:     http.MethodGet,
         wantStatus: http.StatusForbidden,
     },
     ```
   - Remove the `excessive lifetime` case (it no longer applies; the server now computes elapsed time from `time`).
   - Add a `future timestamp allowed` case:
     ```go
     {
         name: "future timestamp allowed",
         makeURL: func() string {
             timestamp := time.Now().Add(time.Hour).Unix()
             return buildProxyURL(proxy.URL, upstream.URL+"/file", timestamp)
         },
         method:     http.MethodGet,
         wantStatus: http.StatusOK,
         wantBody:   "hello proxy",
     },
     ```

- [ ] **Step 2: Run the tests to verify they fail**

Run:
```bash
go test ./internal/proxy/... -v
```

Expected: FAIL because `proxy.go` still reads `expire`.

- [ ] **Step 3: Update `internal/proxy/proxy.go`**

In the `Handler` method, replace:

```go
expireStr := q.Get("expire")
```

with:

```go	timestampStr := q.Get("time")
```

Replace the parameter validation block:

```go
if target == "" || expireStr == "" || sign == "" {
    http.Error(w, "missing parameter", http.StatusBadRequest)
    return
}

var expire int64
_, err := fmt.Sscan(expireStr, &expire)
if err != nil {
    http.Error(w, "invalid expire", http.StatusBadRequest)
    return
}

if !security.Verify(target, expire, sign, p.Config.Secret, p.Config.MaxExpireSeconds) {
```

with:

```go
if target == "" || timestampStr == "" || sign == "" {
    http.Error(w, "missing parameter", http.StatusBadRequest)
    return
}

var timestamp int64
_, err := fmt.Sscan(timestampStr, &timestamp)
if err != nil {
    http.Error(w, "invalid time", http.StatusBadRequest)
    return
}

if !security.Verify(target, timestamp, sign, p.Config.Secret, p.Config.MaxExpireSeconds) {
```

- [ ] **Step 4: Run the tests to verify they pass**

Run:
```bash
go test ./internal/proxy/... -v
```

Expected: PASS.

- [ ] **Step 5: Run the full test suite**

Run:
```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/proxy/proxy.go internal/proxy/proxy_test.go
git commit -m "refactor(proxy): read time parameter and use new security verify"
```

---

## Task 3: Update documentation and helper scripts

**Files:**
- Modify: `README.md`
- Modify: `scripts/test-download.sh`
- Modify: `scripts/test-download.ps1`

- [ ] **Step 1: Update `README.md`**

1. In the "How It Works" diagram, change:
   ```
   GET /download?url=xxx&expire=xxx&sign=xxx
   ```
   to:
   ```
   GET /download?url=xxx&time=xxx&sign=xxx
   ```

2. In the "Security" section, change:
   ```
   signature = HMAC-SHA256(url + expire, secret)
   ```
   to:
   ```
   signature = HMAC-SHA256(url + time, secret)
   ```

- [ ] **Step 2: Update `scripts/test-download.sh`**

1. Replace:
   ```sh
   EXPIRE=$(($(date +%s) + 3600))
   ```
   with:
   ```sh
   TIME=$(date +%s)
   ```

2. Replace:
   ```sh
   SIGN=$(printf 'url=%s&expire=%d' "$TARGET" "$EXPIRE" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $NF}')
   ```
   with:
   ```sh
   SIGN=$(printf 'url=%s&time=%d' "$TARGET" "$TIME" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $NF}')
   ```

3. Replace:
   ```sh
   URL="${PROXY}/download?url=${TARGET_ENC}&expire=${EXPIRE}&sign=${SIGN}"
   ```
   with:
   ```sh
   URL="${PROXY}/download?url=${TARGET_ENC}&time=${TIME}&sign=${SIGN}"
   ```

- [ ] **Step 3: Update `scripts/test-download.ps1`**

1. Replace:
   ```powershell
   $expire = [int64](([DateTime]::UtcNow - $unixEpoch).TotalSeconds + 300)
   ```
   with:
   ```powershell
   $time = [int64](([DateTime]::UtcNow - $unixEpoch).TotalSeconds)
   ```

2. Replace:
   ```powershell
   $payload = "url=${TargetUrl}&expire=${expire}"
   ```
   with:
   ```powershell
   $payload = "url=${TargetUrl}&time=${time}"
   ```

3. Replace:
   ```powershell
   $url = "${ProxyUrl}?url=${targetEncoded}&expire=${expire}&sign=${sign}"
   ```
   with:
   ```powershell
   $url = "${ProxyUrl}?url=${targetEncoded}&time=${time}&sign=${sign}"
   ```

- [ ] **Step 4: Verify script syntax**

For the Bash script:
```bash
sh -n scripts/test-download.sh
```

Expected: no output (success).

For the PowerShell script:
```powershell
powershell -Command "Get-Command scripts/test-download.ps1"
```

Expected: command metadata is printed (file is parseable).

- [ ] **Step 5: Run the Go test suite one final time**

Run:
```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add README.md scripts/test-download.sh scripts/test-download.ps1
git commit -m "docs(scripts): update docs and helpers for time-based signing"
```

---

## Self-Review

1. **Spec coverage:**
   - API parameter change (`expire` → `time`): Task 2.
   - Signature payload change: Task 1.
   - Server-side elapsed-time validation: Task 1.
   - No future-time rejection: Task 1 test and Task 2 test case.
   - README update: Task 3.
   - Helper script updates: Task 3.

2. **Placeholder scan:** No TBD/TODO or vague instructions.

3. **Type consistency:** `Sign` and `Verify` use `timestamp int64` throughout; the handler parses it with `fmt.Sscan` into `int64`; tests pass `int64`.

---

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-07-28-replace-expire-with-time.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
