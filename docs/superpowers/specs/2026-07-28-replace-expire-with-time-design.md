# Design: Replace `expire` with `time` Parameter

## Summary

Change the signed download URL from carrying an absolute expiration timestamp (`expire`) to carrying the request generation timestamp (`time`). The server decides whether the URL is still valid by comparing the elapsed time since `time` against the configured `max_expire_seconds`.

## Motivation

The current `expire` parameter requires the client to decide and compute when a URL should expire. Moving the expiration decision to the server simplifies client logic and makes the configured `max_expire_seconds` the single source of truth for URL lifetime.

## API Change

### Before

```
GET /download?url=<target>&expire=<unix-seconds>&sign=<hmac>
```

### After

```
GET /download?url=<target>&time=<unix-seconds>&sign=<hmac>
```

- `time` is the Unix timestamp at which the signed URL was generated.
- `expire` is no longer accepted.

## Signature Format

The HMAC-SHA256 signature is computed over the literal string:

```
url=<target-url>&time=<unix-seconds>
```

using the configured secret, then hex-encoded.

## Server Validation

In `internal/security.Verify`:

1. Parse `time` from the query string. Reject with `400` if missing or not a valid integer.
2. Compute `elapsed = now - time`.
3. If `elapsed > max_expire_seconds`, reject with `403`.
4. Recompute the expected HMAC over `url=<target-url>&time=<time>` and compare with the provided `sign`. Reject with `403` if they do not match.

The server does **not** reject URLs where `time` is in the future. The only expiration check is the elapsed time since `time`.

## Configuration

`max_expire_seconds` in `config.yaml` keeps the same key but its meaning changes from "maximum distance of `expire` from now" to "maximum lifetime of a signed URL since its `time`".

## Code Changes

- `internal/security/sign.go`
  - Update `Sign` to accept `time int64` and build the new payload.
  - Update `Verify` to validate against the new payload and elapsed-time rule.
- `internal/proxy/proxy.go`
  - Read `time` instead of `expire` from query parameters.
- `internal/proxy/proxy_test.go`
  - Update existing tests and add tests for the new `time`-based behavior.
- `README.md`
  - Update endpoint example and signature description.
- `scripts/test-download.sh`
- `scripts/test-download.ps1`
  - Generate test URLs with `time` instead of `expire`.

## Error Handling

| Condition | Status |
|---|---|
| Missing `url`, `time`, or `sign` | `400 Bad Request` |
| `time` not a valid integer | `400 Bad Request` |
| Elapsed time exceeds `max_expire_seconds` | `403 Forbidden` |
| Signature mismatch | `403 Forbidden` |

## Security Notes

- `time` is included in the HMAC payload, so it cannot be altered by the client without invalidating the signature.
- The server controls the maximum lifetime via `max_expire_seconds`.
- Future `time` values are not rejected; clients can in theory generate URLs with a future timestamp, but the signature still binds them to that exact timestamp and the server-enforced TTL limits the actual usable window.
