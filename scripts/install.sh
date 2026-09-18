#!/bin/bash
set -e

REPO="sepantartd/go-reverse-tunnel"
BINARY_NAME="go-reverse-tunnel"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    i386|i686) ARCH="386" ;;
    *) echo "معماری پشتیبانی نشده: $ARCH"; exit 1 ;;
esac

echo "در حال دریافت آخرین نسخه از $REPO..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "خطا در دریافت آخرین نسخه ریلیز."
    exit 1
fi

FILE_EXT="tar.gz"
if [ "$OS" = "windows" ]; then
    FILE_EXT="zip"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/${BINARY_NAME}_${LATEST_TAG#v}_${OS}_${ARCH}.${FILE_EXT}"

echo "در حال دانلود نسخه $LATEST_TAG برای $OS/$ARCH..."
curl -sL "$DOWNLOAD_URL" -o "tunnel_tmp.${FILE_EXT}"

if [ "$FILE_EXT" = "tar.gz" ]; then
    tar -xzf "tunnel_tmp.${FILE_EXT}" "$BINARY_NAME"
else
    unzip -q "tunnel_tmp.${FILE_EXT}" "$BINARY_NAME.exe"
fi

rm "tunnel_tmp.${FILE_EXT}"
chmod +x "$BINARY_NAME"

if [ -w "/usr/local/bin" ]; then
    mv "$BINARY_NAME" /usr/local/bin/
    echo "پروژه با موفقیت در /usr/local/bin/$BINARY_NAME نصب شد."
else
    echo "پروژه در پوشه جاری آماده است: ./$BINARY_NAME"
fi
