#!/usr/bin/env bash
# Remove cached audio files
# Usage: ./clean-audio-cache.sh [data-dir]
set -euo pipefail

DATA_DIR="${1:-data}"
AUDIO_DIR="$DATA_DIR/audio"

if [ ! -d "$AUDIO_DIR" ]; then
  echo "No audio cache at $AUDIO_DIR"
  exit 0
fi

COUNT=$(find "$AUDIO_DIR" -name "*.m4a" | wc -l | tr -d ' ')
SIZE=$(du -sh "$AUDIO_DIR" 2>/dev/null | cut -f1)

rm -f "$AUDIO_DIR"/*.m4a

echo "Cleared $COUNT files ($SIZE) from $AUDIO_DIR"
