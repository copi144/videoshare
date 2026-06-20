# VideoShare

A self-contained media sharing platform — video, audio, and image — packaged as a single Go binary with an embedded Svelte single-page application. Zero runtime dependencies.

## Features

- **Single binary deployment** — CGO_ENABLED=0 static build. One file, nothing else to install.
- **Multi-user system** — Two roles: `admin` (full control) and `uploader` (upload and delete own media only).
- **TOTP authentication** — No passwords. Login via authenticator app (Google Authenticator, Authy, etc.).
- **Session & token auth** — Session cookie for streaming, Bearer token for API access.
- **Video, audio, and image support** — All media types in one unified system.
- **HLS transcoding** — Adaptive quality ladder (360p, 720p, 1080p) via ffmpeg with concurrent workers.
- **Content-addressed storage** — BLAKE3 hashing deduplicates identical files automatically.
- **Magic-byte detection** — File types identified by content, not extension. Wrong extensions corrected on upload.
- **Password-protected sharing** — Per-resource passwords for share links (`/#/v/{id}`).
- **Categories & playlists** — Admin-created categories assignable to uploaders; playlists group resources within categories.
- **HTTP range request streaming** — Efficient delivery for video and audio.
- **Rate limiting** — 60 requests/minute per IP.
- **Readme support** — Markdown descriptions per resource.
- **In-memory view guard** — Accurate view counting without duplication.

## Quick Start

### Prerequisites

- Go 1.26+ (for backend build)
- Node.js (for frontend build)
- ffmpeg (for HLS transcoding)

### Build and Run

```bash
git clone <repo-url>
cd videoshare

# Build the frontend SPA
cd frontend
npm install
npm run build
cd ..

# Copy the SPA into the backend's embedded assets
cp frontend/dist/index.html backend/web/spa/index.html

# Build the server
cd backend
CGO_ENABLED=0 go build -o ../videoserver .
cd ..

# Run it
./videoserver
```

### First Boot

On first run, the server creates a SQLite database, bootstraps the admin account, and prints a TOTP URI and QR code directly in the terminal:

```
═══════════════════════════════════════════
  Admin Account Created!
  Username: admin
  Scan the QR code below with your
  authenticator app (Google Authenticator, Authy, etc.)
  Or enter the URI manually in your browser
  TOTP URI: otpauth://totp/VideoShare:admin?...
═══════════════════════════════════════════
```

Scan the QR code with your authenticator app, then navigate to `http://localhost:8080`. Enter `admin` as the username and the 6-digit code from your authenticator app.

### Admin TOTP Reset

If you lose access to your authenticator app (phone lost, app reset, etc.), you can reset the admin TOTP secret without losing any data:

1. **Stop the server** if it is running.
2. **Create a file** named `reset-admin-totp.txt` in the `DATA_DIR` directory (default: `./data/`).
3. **Set the admin username** in the file. If the file contains the admin username (e.g., `admin`), that account's TOTP will be reset. If the file is empty, the `ADMIN_USERNAME` environment variable is used instead.
4. **Start the server**. On startup, it will detect the file, generate a new TOTP secret, print the QR code and TOTP URI to the terminal, then **delete the reset file** automatically.
5. **Scan the QR code** with your authenticator app and log in normally.

> ⚠️ The reset file is consumed and deleted on every server startup. If the server fails to read or process the file (e.g., the specified user does not exist), it will log an error and continue without resetting anything.

Example:

```bash
# Stop the server, then:
echo "admin" > data/reset-admin-totp.txt
# Start the server — the new TOTP QR code will be printed in the terminal
./videoserver
```

**Important security note:** Remove the reset file immediately after the server processes it. The server does this automatically, but in the rare event of a crash before the file is deleted, the reset will trigger again on the next startup.

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `:8080` | Listen address (e.g., `:8080` or `127.0.0.1:8080`) |
| `DATA_DIR` | `./data` | Data directory for database and media storage |
| `ADMIN_USERNAME` | `admin` | Admin login username |
| `COOKIE_SECURE` | `false` | Set to `true` when using HTTPS |
| `FFMPEG_PATH` | `ffmpeg` | Path to ffmpeg binary |
| `TRANSCODE_WORKERS` | `1` | Number of concurrent HLS transcodes |

## URL Scheme

### SPA Routes (hash-based)

| Path | Description |
|------|-------------|
| `/#/login` | Login page (TOTP auth) |
| `/#/v/{hash}` | Video/audio/image watch page |
| `/#/v/{hash}/watch` | Watch page (after authentication) |
| `/#/admin` | Admin dashboard (manage users, categories, playlists) |
| `/#/admin/users` | User management (admin only) |
| `/#/admin/categories` | Category management (admin only) |
| `/#/admin/playlists` | Playlist management (admin only) |

### Direct Access Routes

| Path | Description |
|------|-------------|
| `/v/{hash}` | Raw video file (direct access) |
| `/v/{hash}/hls/{path}` | HLS streaming — master.m3u8, segment.ts, etc. |
| `/v/{hash}/download` | Download original file |
| `/a/{hash}` | Audio streaming |
| `/i/{hash}` | Image streaming |

## Authentication Flow

1. **Login**: `POST /api/session` with `{"type": "user", "username": "...", "totp_code": "..."}`
   - Returns an `api_token` (Bearer token) for API calls
   - Sets a session cookie automatically for HLS/streaming access

2. **API calls**: Include `Authorization: Bearer <token>` header on all requests.

3. **Share links**: `POST /api/session` with `{"type": "share", "resource_id": "...", "password": "..."}`
   - Sets a session cookie granting access to HLS streaming for that resource

4. **Token auth**: `POST /api/session` with `{"type": "token", "token": "..."}`
   - Creates a session from an existing Bearer token

5. **HLS streaming**: Requests include the session cookie, validated by the `RequireUserOrVideoAuth` middleware.

## API Reference

### Session Management

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/session` | None | Create session (`type: "user"`, `type: "share"`, or `type: "token"`) |
| DELETE | `/api/session` | Bearer | Destroy current session (logout) |

### Resources

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/resources` | Bearer | List resources (paginated: `?limit=&offset=`) |
| GET | `/api/resources/{id}` | Bearer | Resource detail |
| POST | `/api/upload` | Bearer | Upload media (multipart: `file`, `title`, `description`, `password`, `category_id`) |
| DELETE | `/api/resources/{id}` | Bearer | Delete resource (uploaders can only delete their own) |

### Users

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/me` | Bearer | Current user info |
| GET | `/api/users` | Admin | List users |
| POST | `/api/users` | Admin | Create user (body: `username`, `role`) |

### Categories

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/categories` | Bearer | List categories (paginated) |
| POST | `/api/categories` | Admin | Create category |
| PUT | `/api/categories/{id}` | Admin | Update category |
| DELETE | `/api/categories/{id}` | Admin | Delete category |
| GET | `/api/categories/{id}/uploaders` | Admin | List assigned uploaders |
| POST | `/api/categories/{id}/uploaders` | Admin | Assign uploaders to category |
| DELETE | `/api/categories/{id}/uploaders/{userId}` | Admin | Remove uploader from category |

### Playlists

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/playlists` | Bearer | List playlists (paginated) |
| POST | `/api/playlists` | Admin | Create playlist |
| PUT | `/api/playlists/{id}` | Admin | Update playlist |
| DELETE | `/api/playlists/{id}` | Admin | Delete playlist |
| GET | `/api/playlists/{id}/resources` | Bearer | List resources in playlist |
| POST | `/api/playlists/{id}/resources` | Admin | Add resource to playlist |
| DELETE | `/api/playlists/{id}/resources/{resourceId}` | Admin | Remove resource from playlist |

### Health

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | None | Health check |

## Storage Layout

Media files are stored under `DATA_DIR` organized by type and BLAKE3 hash:

```
DATA_DIR/
├── videoshare.db              # SQLite database
├── video/
│   └── {xx}/
│       └── {yy}/
│           └── {hash}/
│               ├── original   # Original uploaded file
│               ├── thumbnail  # Generated thumbnail
│               └── hls/
│                   ├── 360p/  # 360p HLS segments
│                   ├── 720p/  # 720p HLS segments
│                   ├── 1080p/ # 1080p HLS segments
│                   └── master.m3u8
├── audio/
│   └── {xx}/
│       └── {yy}/
│           └── {hash}/
│               └── original
└── image/
    └── {xx}/
        └── {yy}/
            └── {hash}/
                └── original
```

- `{xx}/{yy}` is derived from the first four hex characters of the BLAKE3 hash (two-level sharding).
- `{hash}` is the full BLAKE3 content hash, used as the resource ID for content-addressed storage.
- Duplicate uploads are detected automatically — identical files produce the same hash and reuse existing storage.

## Tech Stack

**Backend:**
- Go 1.26+
- Chi v5 router
- sqlc (type-safe database queries)
- SCS v2 session manager with SQLite store
- BLAKE3 hashing (t2bot/ahash)
- TOTP (pquerna/otp)
- bcrypt (share passwords)

**Database:**
- SQLite via modernc.org/sqlite (pure Go, CGO-free)
- WAL journal mode for concurrent reads

**Frontend:**
- Svelte 4 SPA
- Tailwind CSS 3
- Vite build tool
- hls.js (HLS playback)
- marked (Markdown rendering)
- QRCode.js (TOTP setup QR display)

**Transcoding:**
- ffmpeg with libx264 and AAC
- HLS adaptive quality ladder (360p, 720p, 1080p)
- Concurrent transcoding workers

**Build:**
- CGO_ENABLED=0 static binary
- vite-plugin-singlefile (SPA embedded into Go binary at compile time)

## Security Notes

1. **TOTP replaces passwords** — All user accounts authenticate via time-based one-time passwords. The TOTP secret is stored in the database and never logged.
2. **Bearer tokens** — API access uses tokens returned at login. These are independent of session cookies and can be used for automated clients.
3. **Session expiry** — Sessions use 30-minute sliding expiry via SCS v2.
4. **Rate limiting** — 60 requests/minute per IP applies globally to all routes.
5. **HTTPS** — Use a reverse proxy (nginx, Caddy) for TLS termination in production. Set `COOKIE_SECURE=true` when behind HTTPS.
6. **File type validation** — Uploaded files are identified by magic bytes, not file extensions. Incorrect extensions are corrected automatically.
7. **Protect DATA_DIR** — Media files and the database contain all application state. Ensure appropriate filesystem permissions.
8. **Roles enforced server-side** — Admin users have full access. Uploader users can only upload to assigned categories and delete their own resources.
