#!/bin/bash
set -euo pipefail

REPO="q5n/download-proxy"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/download-proxy"
DATA_DIR="/var/lib/download-proxy"
LOG_DIR="/var/log/download-proxy"
USER="download-proxy"

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
    cat > "$CONFIG_DIR/config.yaml" <<'EOF'
listen: ":8001"
secret: "change-this-secret"
max_expire_seconds: 3600
allowed_domains:
  - github.com
  - objects.githubusercontent.com
  - release-assets.githubusercontent.com
  - githubusercontent.com
log_file: /var/log/download-proxy/download-proxy.log
EOF
    echo "Created default config at $CONFIG_DIR/config.yaml"
    echo "IMPORTANT: Change the secret before using in production."
fi

chown -R root:root "$CONFIG_DIR"
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
