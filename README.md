# download-proxy

A lightweight and secure HTTP download proxy written in Go.

`download-proxy` allows clients to download files through your own
domain while hiding the original download source. It supports signed
URLs, HTTP redirect following, streaming response forwarding, and
domain-based access control.

## Features

-   🔗 Signed download URLs using HMAC-SHA256
-   🔒 Expiration time validation to prevent unauthorized usage
-   🔄 Automatically follows HTTP redirects (3xx)
-   🚀 Streams file content directly without buffering
-   📦 Supports large files with low memory usage
-   ⏩ Supports HTTP Range requests for resumable downloads
-   🛡️ Domain whitelist protection to reduce SSRF risks
-   ⚡ Lightweight and easy to deploy

## How It Works

    Client
      |
      | GET /download?url=xxx&expire=xxx&sign=xxx
      |
    download-proxy
      |
      | Validate signature
      |
      | Request target URL
      |
      | Follow redirects
      |
      | Stream response
      |
    Client

## Use Cases

-   Proxy GitHub Release downloads through your own domain
-   Provide stable download URLs for software releases
-   Hide upstream storage URLs
-   Build a lightweight software distribution gateway
-   Add access control to public download resources

## Example

Instead of exposing:

    https://github.com/user/project/releases/download/v1.0/app.zip

Provide:

    https://download.example.com/download?url=<signed-url>

The proxy fetches the file from the upstream server and streams it back
to the client.

## Scripts

-   `scripts/install.sh` — One-command install on Debian 12. Downloads the
    latest release binary, creates the `download-proxy` user, installs
    `/etc/download-proxy/config.yaml`, and starts the systemd service.
-   `scripts/release.sh` — Bump version, commit, push, and create a new
    semver tag. Run with `+001`, `+010`, or `+100` to bump
    patch, minor, or major respectively.

## Deployment

See [`docs/nginx.md`](docs/nginx.md) for an nginx reverse proxy + SSL
termination configuration.

## Security

The proxy uses HMAC-SHA256 signed URLs:

    signature = HMAC-SHA256(url + expire, secret)

Each download URL can have:

-   expiration time
-   allowed domains
-   controlled access permissions

## License

MIT License
