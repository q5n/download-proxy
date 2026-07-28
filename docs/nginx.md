# Nginx Reverse Proxy for download-proxy

This document describes how to put nginx in front of `download-proxy` for SSL termination and reverse proxying.

## Assumptions

- `download-proxy` is running on `127.0.0.1:8001`.
- You have a TLS certificate and private key for your domain.
- You are using Debian 12 or a compatible distribution.

## Nginx Configuration

Create or edit `/etc/nginx/sites-available/download-proxy`:

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name download.example.com;

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name download.example.com;

    ssl_certificate /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Enable the site:

```bash
ln -s /etc/nginx/sites-available/download-proxy /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

## Notes

- Replace `download.example.com` and certificate paths with your actual values.
- Ensure `download-proxy` is running before reloading nginx.
- For Let's Encrypt certificates, point `ssl_certificate` and `ssl_certificate_key` to the certbot-generated files.
