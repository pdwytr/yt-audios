#!/bin/bash
# Usage: ./get-channel-id.sh https://www.youtube.com/@ChannelHandle
if [ -z "$1" ]; then
  echo "Usage: $0 <youtube-channel-url>"
  exit 1
fi
yt-dlp --flat-playlist --playlist-items 1 -J "$1" 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('channel_id') or d.get('id'))"
