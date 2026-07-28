# Agent Notes

## Project

Single Go binary (`main.go`) that proxies signed download requests. Run with `go run .` or `go build .`.

## Dependencies

- Go 1.26.4+ (see `go.mod`)
- Only external dep: `gopkg.in/yaml.v3`

## Runtime

- Loads `config.yaml` from the current working directory at startup (not embedded).
- Endpoint: `GET /download?url=<target>&time=<unix>&sign=<hmac>`.
- `config.yaml` ships with a placeholder `secret`; change it before deploying.

## Signature format (code is source of truth)

The HMAC is computed over the literal string:

```
url=<target-url>&time=<unix-seconds>
```

using HMAC-SHA256 with the configured secret, then hex-encoded. This differs slightly from the shorthand in `README.md`.

## Validation behavior

- Signature, `time`, and `url` are all required query parameters.
- `time` is the Unix timestamp when the URL was generated. The request is rejected if `now - time` exceeds `max_expire_seconds`.
- The target URL's hostname must exactly match or be a subdomain of an entry in `allowed_domains`.

## Proxy behavior

- Follows up to 10 HTTP redirects; each redirect target's hostname must also pass `allowed_domains`.
- Rejects the request with 405 if the method is not `GET` or `HEAD`.
- Forwards the incoming request headers to the upstream.
- Copies upstream response headers except `Connection`, `Transfer-Encoding`, and `Keep-Alive`.
- Streams the response body without buffering; logs stream copy errors.

## Server timeouts

`main.go` configures `http.Server` with:

- `ReadTimeout`: 30s
- `WriteTimeout`: 5m
- `IdleTimeout`: 2m

## Testing

Tests exist in `internal/security` and `internal/proxy`. Run `go test ./...` to execute them.
