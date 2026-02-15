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

## Docker Deployment

### 1. Export cookies

```bash
./export-cookies.sh           # defaults to Chrome
./export-cookies.sh firefox   # or specify browser
```

This creates `cookies.txt` in the project root. The cookies file **must** be writable (not read-only) since yt-dlp updates it at runtime.

### 2. Build and start

```bash
docker compose up --build -d
```

The app will be at `http://localhost:8080`. Data persists in `./data/`.

### 3. (Optional) Cloudflare Tunnel

To expose the app via Cloudflare Tunnel:

```bash
export CLOUDFLARE_TUNNEL_TOKEN=your-token-here
docker compose up --build -d
```

### Refreshing cookies

When you see "bot detected" errors, just re-export cookies — no restart needed. Each yt-dlp call reads the file fresh from disk, and the bot detection banner auto-clears after 30 minutes.

```bash
./export-cookies.sh
```

### VPS deployment

**Initial setup** — clone the repo, copy data and cookies, then start:

```bash
# clone on server
ssh user@server "git clone <repo-url> ~/yt-audios"

# copy data and cookies from local machine
scp -r data/ user@server:~/yt-audios/data/
scp cookies.txt user@server:~/yt-audios/cookies.txt

# start on server
ssh user@server "cd yt-audios && docker compose up --build -d"
```

**Updating** — pull latest code and rebuild:

```bash
ssh user@server "cd yt-audios && git pull && docker compose up --build -d"
```

**Cookie refresh** — YouTube cookies expire every few hours. Re-export locally and push to the server. No restart needed, yt-dlp reads the file fresh on every call.

```bash
./export-cookies.sh
scp cookies.txt user@server:~/yt-audios/cookies.txt
```

To automate this, set up a cron job on your local machine (requires browser to be running):

```bash
# crontab -e
0 */3 * * * cd ~/path/to/yt-audios && ./export-cookies.sh && scp cookies.txt user@server:~/yt-audios/cookies.txt
```

## How It Works

1. `/audio?url=<youtube_url>` — runs yt-dlp to extract metadata (title, duration, video ID) and starts downloading the audio file in the background
2. `/stream?id=<video_id>` — serves the downloaded audio file with full seeking support
3. Audio files are cached in a temp directory and cleaned up on server restart
