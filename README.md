# VideoShare

A single-binary, multi-user video sharing server with password-protected sharing,
categories, and playlists — written in Go.

**Key Features:**

- **Single binary** — one file, zero runtime dependencies
- **Multi-user system** — two roles: `admin` (full control) and `uploader` (upload
  and delete own videos only)
- **Login / Logout** — session-based authentication (24 h expiry) with TOTP
  (Time-based One-Time Password) — no passwords needed
- **Categories** — admin-created video categories, assignable to specific uploaders
- **Playlists** — sub-groups within categories; videos can belong to zero or more
  playlists
- **Unassigned videos** — videos not in any playlist shown separately on the
  management page
- **Password-protected sharing** — per-video passwords for share links
- **MP4 streaming** — HTTP range requests for efficient video delivery
- **Rate limiting** — global (60 req/min/IP) and strict for share pages
  (5 req/min/IP)
- **SQLite storage** — no external database required; WAL mode for concurrency
- **Docker support** — multi-stage build, minimal alpine runtime image

## Quick Start

### Build from source

```bash
git clone <repo-url>
cd videoshare
CGO_ENABLED=0 go build -o videoserver ./cmd/server
./videoserver
```

The first time you run it, the server:
1. Creates a SQLite database in `./data/`
2. Bootstraps an admin user with your configured credentials (or defaults)
3. Creates all necessary database tables

### First-run admin setup

On first boot, the server creates an admin account and displays a TOTP URI and
QR code directly in the terminal:

```text
═══════════════════════════════════════════
  Admin Account Created!
  Username: admin
  Scan the QR code below with your
  authenticator app (Google Authenticator, Authy, etc.)
  Or enter the URI manually in your browser
  TOTP URI: otpauth://totp/VideoShare:admin?...
═══════════════════════════════════════════
```

Scan the QR code with any authenticator app, then visit
`http://localhost:8080/login`, enter `admin` as the username and the 6-digit
code from your authenticator app. You'll land on the admin dashboard — ready to
create categories, add uploaders, and start uploading videos.

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `:8080` | Listen address (e.g., `:8080` or `127.0.0.1:8080`) |
| `DATA_DIR` | `./data` | Data directory for SQLite DB and video storage |
| `ADMIN_USERNAME` | `admin` | Admin login username |
| `COOKIE_SECURE` | `false` | Set to `true` when using HTTPS |
| `FFMPEG_PATH` | `ffmpeg` | Path to ffmpeg binary |
| `TRANSCODE_WORKERS` | `1` | Number of concurrent transcodes |

## Docker Deployment

```bash
# Build the image
docker build -t videoserver .

# Run with persistent data
docker run -d \
  --name videoserver \
  -p 8080:8080 \
  -e ADMIN_USERNAME=admin \
  -v ./data:/app/data \
  videoserver
```

### Docker Compose

```yaml
services:
  videoserver:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ADMIN_USERNAME=admin
    volumes:
      - ./data:/app/data
    restart: unless-stopped
```

## API Reference

### Public endpoints (no authentication)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/login` | Login page |
| POST | `/login` | Login with username + TOTP code |
| GET | `/s/{id}` | Video share page (password entry) |
| POST | `/s/{id}/auth` | Verify share password |

### Authenticated endpoints (session required)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/` | — | Redirects to `/admin` |
| GET | `/admin` | User | Main management page (videos, upload form, categories, playlists, unassigned videos) |
| POST | `/api/upload` | User | Upload video (multipart: `file`, `title`, `description`, `password`, `category_id`) |
| POST | `/api/resource/{id}` | User | Delete video (uses `_method=DELETE`; uploaders can only delete their own) |
| POST | `/logout` | User | Logout |

### Admin-only endpoints (admin role required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/categories` | Category management page |
| POST | `/admin/categories` | Create category |
| POST | `/admin/categories/{id}/delete` | Delete category (uses `_method=DELETE`) |
| POST | `/admin/categories/{id}/uploaders` | Assign uploaders to category |
| GET | `/admin/playlists` | Playlist management page |
| POST | `/admin/playlists` | Create playlist |
| POST | `/admin/playlists/{id}/delete` | Delete playlist (uses `_method=DELETE`) |
| POST | `/admin/playlists/{id}/videos` | Add video to playlist |
| POST | `/admin/playlists/{id}/videos/remove` | Remove video from playlist |

### Video streaming

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/video/{id}` | User or share-auth | Video stream with HTTP range request support |

## Security Notes

1. **TOTP replaces passwords** — admin and uploader accounts use TOTP
   (authenticator app). Scan the QR code shown on first boot or when creating
   uploaders. The TOTP secret is stored in the database and never logged.
2. **No external keys needed** — all configuration is via environment variables.
3. **Use a reverse proxy** (nginx, Caddy) for HTTPS termination in production.
4. **Set `COOKIE_SECURE=true`** when using HTTPS.
5. **Roles are enforced server-side** — `admin` users have full access;
   `uploader` users can only upload videos to their assigned categories and
   delete videos they own.
6. Video files are stored in `DATA_DIR/videos/` — protect this directory.
7. The server has built-in rate limiting: 60 requests/minute/IP globally,
   5 password attempts/minute/IP on share pages.

## Tech Stack

- **Backend:** Go 1.25+, Chi v5 router, Go html/template
- **Database:** modernc.org/sqlite (pure Go, CGO-free), WAL journal mode
- **Sessions:** SCS v2 session manager with SQLite store (24 h expiry)
- **Auth:** TOTP (pquerna/otp)
- **Frontend:** Pico CSS v2 (classless, semantic HTML framework)
- **Build:** `CGO_ENABLED=0` static binary, multi-stage Docker build
