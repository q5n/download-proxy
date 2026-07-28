#!/bin/bash
set -euo pipefail

REPO="q5n/download-proxy"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/download-proxy"
DATA_DIR="/var/lib/download-proxy"
LOG_DIR="/var/log/download-proxy"
USER="download-proxy"

PORT="8001"
SECRET=""

while [ $# -gt 0 ]; do
    case "$1" in
        -p|--port)
            PORT="${2:-}"
            if [ -z "$PORT" ]; then
                echo "Option $1 requires an argument" >&2
                exit 1
            fi
            shift 2
            ;;
        -s|--secret)
            SECRET="${2:-}"
            if [ -z "$SECRET" ]; then
                echo "Option $1 requires an argument" >&2
                exit 1
            fi
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [-p|--port PORT] [-s|--secret SECRET]"
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            echo "Usage: $0 [-p|--port PORT] [-s|--secret SECRET]" >&2
            exit 1
            ;;
    esac
done

if ! [[ "$PORT" =~ ^[0-9]+$ ]]; then
    echo "Invalid port: $PORT" >&2
    exit 1
fi

if [ -z "$SECRET" ]; then
    printf 'Enter secret: ' >&2
    read -r SECRET
fi

if [ -z "$SECRET" ]; then
    echo "secret is required" >&2
    exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root" >&2
    exit 1
fi

echo "Creating user and directories..."
if ! id -u "$USER" >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin "$USER"
fi

mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
chown -R "$USER:$USER" "$DATA_DIR" "$LOG_DIR"

echo "Downloading latest release..."
LATEST_URL=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | \
    grep -o '"browser_download_url": "[^"]*download-proxy-linux-amd64"' | \
    cut -d'"' -f4)

if [ -z "$LATEST_URL" ]; then
    echo "Failed to find release binary" >&2
    exit 1
fi

curl -fsSL -o "$INSTALL_DIR/download-proxy" "$LATEST_URL"
chmod +x "$INSTALL_DIR/download-proxy"

if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    cat > "$CONFIG_DIR/config.yaml" <<EOF
listen: ":${PORT}"
secret: "${SECRET}"
max_expire_seconds: 3600
allowed_domains:
  - github.com
  - objects.githubusercontent.com
  - release-assets.githubusercontent.com
  - githubusercontent.com
log_file: /var/log/download-proxy/download-proxy.log
EOF
    echo "Created default config at $CONFIG_DIR/config.yaml"
fi

chown -R root:download-proxy "$CONFIG_DIR"
chmod 640 "$CONFIG_DIR/config.yaml"

cat > /etc/systemd/system/download-proxy.service <<EOF
[Unit]
Description=download-proxy
After=network.target

[Service]
Type=simple
User=$USER
Group=$USER
WorkingDirectory=$CONFIG_DIR
ExecStart=$INSTALL_DIR/download-proxy
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now download-proxy

echo "download-proxy installed and started."
echo "Check status: systemctl status download-proxy"
