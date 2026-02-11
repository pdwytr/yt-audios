# CLAUDE.md

## Project Structure

- `main.go` — Go backend: API endpoints (`/audio`, `/stream`), embeds built frontend via `//go:embed all:frontend/dist`
- `frontend/` — React + TypeScript + Vite SPA
  - `src/App.tsx` — Single component, the entire player UI
  - `src/App.css` — All styles
  - `vite.config.ts` — Dev proxy: `/audio` and `/stream` forward to Go on :8080

## Build

- Frontend: `cd frontend && npm run build` (outputs to `frontend/dist/`)
- Full binary: `make build` (builds frontend, then `go build`)
- `frontend/dist/` must exist before `go build` (the embed directive requires it)

## Architecture

- yt-dlp is called twice per video: once for metadata (`--print`), once to download the file (`-o`)
- Downloaded audio files are cached in a temp dir (`os.MkdirTemp`), cleaned up on server restart
- `handleStream` uses `http.ServeFile` which handles Range requests automatically (seeking works)
- Background downloads use `sync.Map` with channels to coordinate: `/stream` waits for download to finish before serving

## Key Decisions

- Audio is downloaded to disk (not streamed/proxied from YouTube URLs) because YouTube throttles direct URL access (~32 KB/s) but yt-dlp's own downloader gets full speed (~8 MB/s)
- No yt-dlp client overrides — the default client selection (`android_vr` + `web_safari`) is the most reliable across all videos
