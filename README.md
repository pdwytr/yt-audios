# YT Audios

A self-hosted web player that turns YouTube videos and playlists into audio you can listen to in the browser. One Go binary, embedded React UI, powered by yt-dlp.

## What it does

Paste a YouTube URL and listen. The Go backend uses yt-dlp to read metadata and download the audio track to a local cache, then streams it with full seeking support. Signed-in users can also save playlists and resume where they left off.

## Features

- Audio-only playback of any YouTube video URL, with seek support (HTTP range requests)
- Playback speed control (1x to 1.75x), next/previous track, remembered speed
- Playlists (up to 3 per user) with per-playlist resume position, saved every 10 seconds and on pause/close
- Simple accounts: register/login with session tokens, one active session per user
- Optional channel allow-list (`data/channels.csv`); leave it empty to allow any channel
- On-disk audio cache; downloads run in the background and `/stream` waits for them
- Bot-detection awareness: surfaces YouTube blocking and recovers when cookies are refreshed
- Startup data-schema validation (`-validate-data` for a dry run)
- Single binary with the frontend embedded; Dockerfile and docker-compose included

## Tech stack

- Backend: Go (standard library only), yt-dlp for extraction
- Frontend: React 19, TypeScript, Vite
- Storage: CSV and JSON files under `data/` (no database)

## Prerequisites

- [Go](https://go.dev/dl/) 1.25+
- [Node.js](https://nodejs.org/) 20+ and npm
- [yt-dlp](https://github.com/yt-dlp/yt-dlp#installation) on your `PATH`, plus [ffmpeg](https://ffmpeg.org/) and a JavaScript runtime (Node works) for current YouTube extraction

## Quickstart

```bash
git clone https://github.com/pdwytr/yt-audios.git
cd yt-audios
make build        # builds the frontend, then the Go binary
./yt-audios       # serves on http://localhost:8080
```

Open http://localhost:8080, register an account, then add a playlist, or call the API directly:

```bash
curl "http://localhost:8080/audio?url=https://www.youtube.com/watch?v=VIDEO_ID"
```

## How the one-binary build works

1. `make build` runs `npm run build` in `frontend/`, producing `frontend/dist/`.
2. `main.go` embeds that folder with `//go:embed all:frontend/dist`.
3. `go build -o yt-audios .` compiles the server and the UI into one executable.

`frontend/dist/` must exist before `go build` (the embed fails otherwise). The only runtime dependency is `yt-dlp` on your `PATH`.

## Usage

### Configuration

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP port |
| `DATA_DIR` | `data` | Where CSV/JSON state and the audio cache live |
| `YT_COOKIES` | unset | Path to a Netscape `cookies.txt` passed to yt-dlp |
| `YT_BROWSER` | unset | Browser name for `--cookies-from-browser` (used if `YT_COOKIES` is unset) |

### HTTP API

| Endpoint | Auth | Description |
|---|---|---|
| `GET /audio?url=` | no | Returns `{video_id, title, duration}` and starts caching audio |
| `GET /stream?id=` | no | Streams cached audio (supports `Range`) |
| `POST /api/register`, `POST /api/login` | no | Returns a bearer token |
| `GET /api/register/info` | no | Remaining registration slots (max 40 users) |
| `GET /api/channels` | no | Allowed channels |
| `GET/POST/DELETE /api/playlists` | yes | List, add (`{"url"}`), remove (`?id=`) playlists |
| `GET /api/playlists/videos?id=` | yes | Videos of a playlist |
| `POST /api/playlists/refresh?id=` | yes | Re-fetch a playlist |
| `GET/POST /api/progress` | yes | Read/save playback position |
| `GET /api/status` | no | Bot-detection status |

### Cookies (recommended)

YouTube may throttle or block unknown IPs. Export browser cookies so yt-dlp can authenticate:

```bash
./export-cookies.sh            # Chrome by default
./export-cookies.sh firefox
YT_COOKIES=cookies.txt ./yt-audios
```

The cookies file must be writable (yt-dlp updates it) and is git-ignored. yt-dlp reads it on every call, so refreshing cookies needs no restart. When the UI reports "bot detected", re-export; the banner clears after 30 minutes.

### Helper scripts

- `export-cookies.sh [browser]` - export YouTube/Google cookies to `cookies.txt`
- `get-channel-id.sh <channel-url>` - find a channel ID for `data/channels.csv`
- `clean-audio-cache.sh [data-dir]` - delete cached `.m4a` files

### Development

```bash
make dev-backend    # Go server on :8080
make dev-frontend   # Vite dev server on :5173, proxies API calls to :8080
```

The Go server needs `frontend/dist/` to exist once (`cd frontend && npm ci && npm run build`) so it compiles.

### Docker

```bash
./export-cookies.sh            # on a machine with a browser logged in to YouTube
docker compose up --build -d   # app on http://localhost (port 80 -> 8080)
```

`docker-compose.yml` also defines an optional Cloudflare Tunnel service; set `CLOUDFLARE_TUNNEL_TOKEN` to use it, or remove that service. Data persists in `./data/`.

To deploy to a server, clone the repo there, copy `data/` and `cookies.txt` from your machine, and run `docker compose up --build -d`. To update: `git pull && docker compose up --build -d`.

## Data files

Validated at startup; run `go run . -validate-data` to check without starting the server.

- `data/users.csv`: `username,password`
- `data/channels.csv`: `id,name` (optional allow-list)
- `data/playlists.csv`: `username,playlist_id,url,title`
- `data/videos.csv`: `playlist_id,video_id,title,duration,position`
- `data/progress/*.json`: per-user playback positions

Note: passwords are stored in plaintext in `users.csv`. Use for personal or trusted-group deployments only.

## Testing

```bash
go vet ./... && go test ./...
cd frontend && npm run lint && npm run build
```
