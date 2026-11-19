# CLAUDE.md

## Project Structure

- `main.go` — Entry point, embeds frontend (`//go:embed all:frontend/dist`), audio download/stream handlers, bot detection, cookie handling
- `handlers.go` — API handlers: playlists CRUD, playlist video fetching, channel listing, progress tracking, channel validation
- `store.go` — CSV/JSON persistence: channels, playlists, videos, per-user progress. In-memory caches with RWMutex, atomic file writes
- `auth.go` — Session-based auth: login/register handlers, Bearer token middleware, users.csv storage (plaintext passwords, max 40 users)
- `frontend/` — React + TypeScript + Vite SPA
  - `src/App.tsx` — Router & navigation (views: login, playlists, videos, player)
  - `src/api.ts` — HTTP client with auth wrapper, all API calls
  - `src/types.ts` — TypeScript interfaces
  - `src/components/Login.tsx` — Login/register form
  - `src/components/PlaylistList.tsx` — Playlist management, channel list, add/delete playlists
  - `src/components/VideoList.tsx` — Video listing with duration formatting
  - `src/components/Player.tsx` — Audio player: seek, speed control, auto-save progress, next/prev
  - `src/App.css` — Dark theme styles
  - `vite.config.ts` — Dev proxy: `/audio`, `/stream`, `/api` forward to Go on :8080

## Scripts

- `scripts/export-cookies.sh [browser]` — Export YouTube cookies from browser (default: chrome) to `cookies.txt`
- `scripts/get-channel-id.sh <channel-url>` — Get YouTube channel ID for whitelisting
- `scripts/clean-audio-cache.sh [data-dir]` — Delete cached .m4a files from data/audio/

## Data Files

- `data/channels.csv` — Channel whitelist (id, name). Only playlists from these channels can be added
- `data/users.csv` — User credentials (username, password). Plaintext
- `data/playlists.csv` — User-playlist associations (username, playlist_id, url, title)
- `data/videos.csv` — Cached video metadata (playlist_id, video_id, title, duration, position)
- `data/progress/{username}.json` — Per-user playback positions (global + per-playlist)
- `data/audio/` — Cached downloaded .m4a files

## Build & Deploy

- Frontend: `cd frontend && npm run build` (outputs to `frontend/dist/`)
- Full binary: `make build` (builds frontend, then `go build`)
- Dev: `make dev-backend` + `make dev-frontend` (two terminals)
- Docker: `docker compose up --build -d` (multi-stage build, port 80 -> 8080)
- `frontend/dist/` must exist before `go build` (embed directive requires it)

## Architecture

- yt-dlp called twice per video: once for metadata (`--print`, 30s timeout), once for download (`-o`, 120s timeout)
- Audio cached in `data/audio/{videoID}.m4a`, persists across restarts
- `handleStream` uses `http.ServeFile` — Range requests (seeking) handled automatically
- Background downloads use `sync.Map` with channels: `/stream` waits for download to finish before serving
- CSV storage with in-memory caches, atomic writes (temp file + rename)
- Sessions in-memory only (lost on restart)
- Bot detection: scans yt-dlp stderr for "Sign in to confirm" / "HTTP Error 429", auto-clears after 30 min
- Channel whitelist enforced on playlist add — validates via yt-dlp `--flat-playlist`
- Progress auto-saved every 10s during playback, on pause, and via beacon on page unload
- Playlists limited to 3 per user

## Key Decisions

- Audio downloaded to disk (not proxied) because YouTube throttles direct URL access (~32 KB/s) but yt-dlp gets full speed (~8 MB/s)
- No yt-dlp client overrides — default client selection is most reliable
- Cookie auth (`YT_COOKIES` env var) bypasses YouTube rate limiting
- Embedded frontend in Go binary — single binary deployment
- Cloudflare Tunnel (optional) for public HTTPS access via docker-compose tunnel service
