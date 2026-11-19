# data/

Runtime storage for the server (no database). Override the location with `DATA_DIR`.

Tracked scaffolds (header row only):

- `users.csv` - `username,password` (plaintext; keep this deployment private)
- `channels.csv` - `id,name` optional channel allow-list; empty means any channel is allowed
- `playlists.csv` - `username,playlist_id,url,title`
- `videos.csv` - `playlist_id,video_id,title,duration,position`

Created at runtime and git-ignored:

- `audio/` - cached `.m4a` downloads (clear with `scripts/clean-audio-cache.sh`)
- `progress/` - per-user playback positions as `{username}.json`

Find a channel ID for the allow-list with `scripts/get-channel-id.sh <channel-url>`.
