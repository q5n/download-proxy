#!/bin/bash

PROXY="http://127.0.0.1:8001"
TARGET=""
OUTPUT=""

while [ $# -gt 0 ]; do
    case "$1" in
        -o|--output)
            OUTPUT="${2:-}"
            if [ -z "$OUTPUT" ]; then
                echo "Option $1 requires an argument" >&2
                exit 1
            fi
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 <target-url> [-o|--output <output-path>]"
            exit 0
            ;;
        -*)
            echo "Unknown option: $1" >&2
            echo "Usage: $0 <target-url> [-o|--output <output-path>]" >&2
            exit 1
            ;;
        *)
            if [ -z "$TARGET" ]; then
                TARGET="$1"
            else
                echo "Unexpected argument: $1" >&2
                exit 1
            fi
            shift
            ;;
    esac
done

if [ -z "$TARGET" ]; then
    echo "Usage: $0 <target-url> [-o|--output <output-path>]" >&2
    exit 1
fi

if [ -z "$OUTPUT" ]; then
    OUTPUT="download-$(date +%Y%m%d-%H%M%S).tmp"
fi

if [ -n "$DOWNLOAD_PROXY_SECRET" ]; then
    SECRET="$DOWNLOAD_PROXY_SECRET"
else
    printf 'Enter secret: ' >&2
    read -s -r SECRET
fi

if [ -z "$SECRET" ]; then
    echo "secret is required" >&2
    exit 1
fi

TIME=$(date +%s)

# 纯 sh URL 编码
url_encode() {
    local input="$1"
    local output=""
    local i char hex
    for (( i=0; i<${#input}; i++ )); do
        char="${input:i:1}"
        case "$char" in
            [a-zA-Z0-9.~_-])
                output+="$char"
                ;;
            *)
                hex=$(printf '%02x' "'$char")
                output+="%$hex"
                ;;
        esac
    done
    printf '%s' "$output"
}

# HMAC-SHA256 签名
SIGN=$(printf 'url=%s&time=%d' "$TARGET" "$TIME" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $NF}')

TARGET_ENC=$(url_encode "$TARGET")
URL="${PROXY}/download?url=${TARGET_ENC}&time=${TIME}&sign=${SIGN}"

echo "Request URL: $URL"
echo "Output file: $OUTPUT"
curl -v -L -o "$OUTPUT" "$URL"
