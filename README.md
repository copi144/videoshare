# VideoShare

A single-binary, password-protected video sharing server written in Go.

**Key Features:**
- Single binary deployment — one file, zero dependencies
- Password-protected video sharing (per-video passwords)
- Session-based authentication with 24h expiry
- SQLite storage (no external database needed)
- CSRF protection + rate limiting
- MP4 streaming with HTTP range requests
- Docker support

## Quick Start

### One-line run

```bash
go install github.com/yourusername/videoshare/cmd/server@latest
videoserver
```

The first time you run it, a random upload password and session key are generated
and printed to the logs. **Copy the upload password** — you'll need it to upload videos.

### Build from source

```bash
git clone <repo-url>
cd videoshare
CGO_ENABLED=0 go build -o videoserver ./cmd/server
./videoserver
```

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `:8080` | Listen address (e.g., `:8080` or `127.0.0.1:8080`) |
| `DATA_DIR` | `./data` | Data directory for SQLite DB and video storage |
| `UPLOAD_PASSWORD` | *(random)* | Admin password for uploading videos |
| `SESSION_KEY` | *(random)* | Session encryption key (set for persistence across restarts) |
| `CSRF_KEY` | *(random)* | CSRF protection key (set for persistence across restarts) |
| `COOKIE_SECURE` | `false` | Set to `true` when using HTTPS |

> If `UPLOAD_PASSWORD`, `SESSION_KEY`, or `CSRF_KEY` are not set, random values
> are generated at startup and logged. Set them explicitly for persistent sessions.

## Docker Deployment

```bash
# Build the image
docker build -t videoserver .

# Run with persistent data
docker run -d \
  --name videoserver \
  -p 8080:8080 \
  -e UPLOAD_PASSWORD=mysecretpassword \
  -e SESSION_KEY=$(openssl rand -hex 32) \
  -e CSRF_KEY=$(openssl rand -hex 32) \
  -v ./data:/app/data \
  videoserver
```

### Docker Compose

```yaml
version: '3'
services:
  videoserver:
    build: .
    ports:
      - "8080:8080"
    environment:
      - UPLOAD_PASSWORD=mysecretpassword
      - SESSION_KEY=change-me-32-char-hex-key
      - CSRF_KEY=change-me-too-32-char-hex-key
    volumes:
      - ./data:/app/data
    restart: unless-stopped
```

## Security Notes

1. **Always set `UPLOAD_PASSWORD`** to a strong password in production
2. **Set `SESSION_KEY` and `CSRF_KEY`** to persistent random values to maintain sessions across restarts
3. **Use a reverse proxy** (nginx, Caddy) for HTTPS termination in production
4. **Set `COOKIE_SECURE=true`** when using HTTPS
5. Video files are stored in `DATA_DIR/videos/` — protect this directory
6. The server has built-in rate limiting (5 password attempts/minute/IP)

## API Reference

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | None | Health check |
| GET | `/login` | None | Unauthorized page |
| GET | `/s/{id}` | None | Password entry page |
| POST | `/s/{id}/auth` | None | Verify password |
| GET | `/s/{id}/watch` | Session | Video player page |
| GET | `/admin` | Session | Upload management |
| POST | `/api/upload` | Session | Upload video file |
| POST | `/api/resource/{id}` | Session | Delete video (uses `_method=DELETE`) |
| GET | `/api/video/{id}` | Session | Video stream (range requests) |

## Tech Stack

- **Backend:** Go 1.24+, Chi router, modernc.org/sqlite
- **Frontend:** Go html/template, Pico CSS v2
- **Build:** CGO_ENABLED=0 static binary
- **Database:** SQLite (WAL mode)
