#!/bin/bash

# DlSecret / DlProxy / DlTarget 均可通过同名环境变量或命令行参数提供：
#   DlSecret=<secret> DlProxy=<proxy-url> DlTarget=<target-url> ./test-download.sh
#   ./test-download.sh --secret <secret> --proxy <proxy-url> --target <target-url> [-o <output>]
# 命令行参数优先于环境变量；两者都未提供时报错退出。
# 兼容旧变量名 DOWNLOAD_PROXY_SECRET 作为 secret 的兜底。

usage() {
    cat >&2 <<EOF
Usage: $0 [-s|--secret <secret>] [-p|--proxy <proxy-url>] [-t|--target <target-url>] [-o|--output <output-path>]
       $0 <target-url> [-o|--output <output-path>]

Options (or environment variables):
  -s, --secret   HMAC secret            (env: DlSecret)
  -p, --proxy    proxy endpoint URL     (env: DlProxy), e.g. https://your-domain.com/download
  -t, --target   target download URL    (env: DlTarget)
  -o, --output   output file path       (default: basename of target URL)
  -h, --help     show this help
EOF
}

SECRET="${DlSecret:-${DOWNLOAD_PROXY_SECRET:-}}"
PROXY="${DlProxy:-}"
TARGET="${DlTarget:-}"
OUTPUT=""

while [ $# -gt 0 ]; do
    case "$1" in
        -s|--secret)
            SECRET="${2:-}"
            if [ -z "$SECRET" ]; then echo "Option $1 requires an argument" >&2; exit 1; fi
            shift 2
            ;;
        -p|--proxy)
            PROXY="${2:-}"
            if [ -z "$PROXY" ]; then echo "Option $1 requires an argument" >&2; exit 1; fi
            shift 2
            ;;
        -t|--target)
            TARGET="${2:-}"
            if [ -z "$TARGET" ]; then echo "Option $1 requires an argument" >&2; exit 1; fi
            shift 2
            ;;
        -o|--output)
            OUTPUT="${2:-}"
            if [ -z "$OUTPUT" ]; then echo "Option $1 requires an argument" >&2; exit 1; fi
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            echo "Unknown option: $1" >&2
            usage
            exit 1
            ;;
        *)
            if [ -z "$TARGET" ]; then
                TARGET="$1"
            else
                echo "Unexpected argument: $1" >&2
                usage
                exit 1
            fi
            shift
            ;;
    esac
done

if [ -z "$SECRET" ]; then
    echo "Error: secret is required (option --secret or env DlSecret)" >&2
    usage
    exit 1
fi
if [ -z "$PROXY" ]; then
    echo "Error: proxy is required (option --proxy or env DlProxy)" >&2
    usage
    exit 1
fi
if [ -z "$TARGET" ]; then
    echo "Error: target is required (option --target, positional argument, or env DlTarget)" >&2
    usage
    exit 1
fi

if [ -z "$OUTPUT" ]; then
    OUTPUT="${TARGET##*/}"
fi
if [ -z "$OUTPUT" ]; then
    OUTPUT="download-$(date +%Y%m%d-%H%M%S).tmp"
fi

TIME=$(date +%s)
NONCE=$(openssl rand -hex 16)

# HMAC-SHA256 签名: url=<target>&time=<unix>&nonce=<nonce>
SIGN=$(printf 'url=%s&time=%d&nonce=%s' "$TARGET" "$TIME" "$NONCE" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $NF}')

echo "Proxy endpoint: $PROXY"
echo "Target URL: $TARGET"
echo "Output file: $OUTPUT"
curl -v -L -G "$PROXY" \
    --data-urlencode "url=$TARGET" \
    --data-urlencode "time=$TIME" \
    --data-urlencode "nonce=$NONCE" \
    --data-urlencode "sign=$SIGN" \
    -o "$OUTPUT"
