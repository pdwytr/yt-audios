# YT Audio Player

A web-based YouTube audio player. Paste a YouTube URL, listen to the audio. Go backend + React frontend, ships as a single binary.

## Prerequisites

- [Go](https://go.dev/dl/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [yt-dlp](https://github.com/yt-dlp/yt-dlp#installation)

## Setup

### Cookies (recommended)

YouTube throttles requests from unknown IPs. Export your browser cookies so yt-dlp can authenticate:

```bash
./export-cookies.sh        # defaults to Chrome
./export-cookies.sh firefox # or specify browser
```

This creates `cookies.txt` with only YouTube/Google cookies. Then run the backend with:

```bash
YT_COOKIES=cookies.txt make dev-backend
```

Re-run the script whenever cookies expire.

### Users

Edit `users.csv` to add login credentials (format: `username,password` per line).

## Development

Two terminals:

```bash
# Terminal 1: Go backend on :8080
YT_COOKIES=cookies.txt make dev-backend

# Terminal 2: Vite dev server on :5173 (proxies API to Go)
make dev-frontend
```

Open http://localhost:5173

## Production Build

```bash
make build
./yt-audios
```

This builds the React frontend, embeds it into the Go binary, and produces a single `yt-audios` executable. Open http://localhost:8080

## Docker

```bash
# 1. Export cookies locally first
./export-cookies.sh

# 2. Run with docker-compose
docker compose up -d
```

The compose file mounts `cookies.txt` and persists data in `./data/`. Edit `users.csv` before building to set up login credentials.

To deploy on a VPS, copy these files over and run `docker compose up -d`:
- `docker-compose.yml`
- `Dockerfile`
- `users.csv`
- `cookies.txt`
- source code (or pre-built image)

Re-run `./export-cookies.sh` locally and `scp cookies.txt` to the VPS when cookies expire.

## How It Works

1. `/audio?url=<youtube_url>` — runs yt-dlp to extract metadata (title, duration, video ID) and starts downloading the audio file in the background
2. `/stream?id=<video_id>` — serves the downloaded audio file with full seeking support
3. Audio files are cached in a temp directory and cleaned up on server restart
