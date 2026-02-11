# YT Audio Player

A web-based YouTube audio player. Paste a YouTube URL, listen to the audio. Go backend + React frontend, ships as a single binary.

## Prerequisites

- [Go](https://go.dev/dl/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [yt-dlp](https://github.com/yt-dlp/yt-dlp#installation)

## Development

Two terminals:

```bash
# Terminal 1: Go backend on :8080
make dev-backend

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

## How It Works

1. `/audio?url=<youtube_url>` — runs yt-dlp to extract metadata (title, duration, video ID) and starts downloading the audio file in the background
2. `/stream?id=<video_id>` — serves the downloaded audio file with full seeking support
3. Audio files are cached in a temp directory and cleaned up on server restart
