#!/usr/bin/env bash
# Export YouTube-relevant cookies from Chrome to cookies.txt
# Usage: ./export-cookies.sh [browser]  (default: chrome)
set -euo pipefail

BROWSER="${1:-chrome}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUT="$SCRIPT_DIR/cookies.txt"
TMP="$SCRIPT_DIR/.cookies.tmp"

echo "Extracting cookies from $BROWSER..."
yt-dlp --cookies-from-browser "$BROWSER" \
  --cookies "$TMP" \
  --no-download "https://www.youtube.com/watch?v=dQw4w9WgXcQ" \
  >/dev/null 2>&1

# Keep only YouTube and Google auth cookies
awk '/^#/ || /^$/ { print; next }
     { split($0, a, "\t"); d = a[1] }
     d ~ /(^|\.)youtube\.com$/ || d ~ /(^|\.)google\.com$/ { print }
' "$TMP" > "$OUT"

rm -f "$TMP"

TOTAL=$(grep -c -v "^#\|^$" "$OUT" || true)
echo "Saved $TOTAL cookies to $OUT"
