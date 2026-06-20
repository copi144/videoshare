# Database Schema — VideoShare v1.0

SQLite database with 11 tables. WAL journal mode for concurrent reads.

## Overview

```
users ──┬── sessions
         ├── api_tokens
         ├── categories ──┬── category_users
         │                 └── resource_categories ──┐
         ├── resources ────resource_categories ──────┘
         │                 └── share_resources
         │                 └── playlist_videos ──┐
         └── playlists ──── playlist_videos ─────┘
         └── share_links
```

## Tables

### 1. `users`

Stores user accounts. Each user authenticates via TOTP (time-based one-time password) — no traditional passwords. Usernames are unique and serve as the primary identifier across the system.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `name` | TEXT | `PRIMARY KEY` | Username (unique identifier) |
| `totp_secret` | TEXT | `NOT NULL` | TOTP secret for authenticator app |
| `display_name` | TEXT | `NOT NULL DEFAULT ''` | Human-readable display name |
| `is_admin` | INTEGER | `NOT NULL DEFAULT 0` | Boolean: 1=admin, 0=regular user |
| `created_at` | DATETIME | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Account creation timestamp |

**Indexes:** Implicit primary key index on `name`.

**Relationships:** Referenced by `sessions.username`, `api_tokens.username`, `categories.created_by`, `category_users.name`, `resources.uploaded_by`, `share_resources.created_by`, `playlists.created_by`, `share_links.created_by`.

---

### 2. `sessions`

Stores active browser sessions for HLS streaming and web access. Uses server-side session management via SCS v2 with SQLite store.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `token` | TEXT | `PRIMARY KEY` | Session token (randomly generated) |
| `data` | BLOB | `NOT NULL` | Encoded session data managed by SCS v2 |
| `expiry` | DATETIME | `NOT NULL` | Session expiration timestamp |

**Indexes:** Implicit primary key index on `token`.

**Relationships:** No foreign key constraints — managed entirely by the SCS v2 session manager. Application code reads and writes sessions by token.

---

### 3. `api_tokens`

Stores Bearer tokens issued at login for API access. Independent of browser sessions — tokens can be used for automated clients and persist until expiry.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `token` | TEXT | `PRIMARY KEY` | Bearer token string |
| `username` | TEXT | `NOT NULL REFERENCES users(name)` | Owner of the token |
| `expires_at` | DATETIME | `NOT NULL` | Token expiration timestamp |
| `created_at` | DATETIME | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Token creation timestamp |

**Indexes:** Implicit primary key index on `token`.

**Foreign Key:** `username` references `users(name)`. Deleting a user leaves orphaned tokens (no `ON DELETE CASCADE`); the application should manage token cleanup on user deletion.

---

### 4. `categories`

Defines content categories for organizing media. Categories group resources and control upload access — regular users must be assigned to a category with upload permission to contribute media.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `name` | TEXT | `PRIMARY KEY` | Category identifier (URL-safe slug) |
| `display_name` | TEXT | `NOT NULL DEFAULT ''` | Human-readable category name |
| `description` | TEXT | `NOT NULL DEFAULT ''` | Category description |
| `created_by` | TEXT | `NOT NULL REFERENCES users(name)` | Admin who created the category |
| `created_at` | DATETIME | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Creation timestamp |

**Indexes:** Implicit primary key index on `name`.

**Foreign Key:** `created_by` references `users(name)`. No cascade — deleting a user that created categories leaves the categories intact.

**Relationships:** Referenced by `category_users.category_name`, `resource_categories.category_name`, `playlists.category_name`, `share_links` (via `target_id` when `target_type = 'category'`).

---

### 5. `category_users`

Maps users to categories with upload permission flags. Implements a many-to-many relationship. A user's `can_upload` flag determines whether they can upload media to that category. Users can be members of multiple categories with independent permissions.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `category_name` | TEXT | `NOT NULL REFERENCES categories(name) ON DELETE CASCADE` | Category name |
| `name` | TEXT | `NOT NULL REFERENCES users(name) ON DELETE CASCADE` | Username |
| `can_upload` | INTEGER | `NOT NULL DEFAULT 0` | Boolean: 1=can upload to this category |

**Primary Key:** Composite `(category_name, name)` — each user appears at most once per category.
**Indexes:** Implicit primary key index on `(category_name, name)`.

**Foreign Keys:**
- `category_name` references `categories(name) ON DELETE CASCADE` — deleting a category removes all its membership records.
- `name` references `users(name) ON DELETE CASCADE` — deleting a user removes all their category memberships.

---

### 6. `resources`

Stores metadata for all uploaded media (video, audio, image). Content is stored on disk by BLAKE3 hash; the `id` column is the full content hash, enabling content-addressed storage and automatic deduplication.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | TEXT | `PRIMARY KEY` | BLAKE3 content hash (used as storage path) |
| `title` | TEXT | `NOT NULL` | Display title |
| `filename` | TEXT | `NOT NULL` | Original filename at upload time |
| `file_size` | INTEGER | `NOT NULL` | File size in bytes |
| `content_type` | TEXT | `NOT NULL` | MIME type (e.g., `video/mp4`, `audio/mpeg`, `image/jpeg`) |
| `resource_type` | TEXT | `NOT NULL` | Media category: `video`, `audio`, or `image` |
| `views` | INTEGER | `NOT NULL DEFAULT 0` | View count (guarded against duplicate counting) |
| `uploaded_by` | TEXT | `NOT NULL REFERENCES users(name)` | Username of the uploader |
| `transcode_status` | TEXT | `NOT NULL DEFAULT 'none'` | HLS transcode status: `none`, `processing`, `done`, or `error` |
| `banned` | INTEGER | `NOT NULL DEFAULT 0` | Boolean: 1=banned (hidden from listings) |
| `no_transcode` | INTEGER | `NOT NULL DEFAULT 0` | Boolean: 1=skip HLS transcoding for this resource |
| `created_at` | DATETIME | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Upload timestamp |
| `updated_at` | DATETIME | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Last update timestamp |

**Indexes:** Implicit primary key index on `id`.

**Foreign Key:** `uploaded_by` references `users(name)`. No cascade — deleting a user preserves their uploaded resources.

**Relationships:** Referenced by `resource_categories.resource_id`, `share_resources.resource_id`, `playlist_videos.resource_id`.

---

### 7. `resource_categories`

Maps resources to categories. Implements a many-to-many relationship — a resource can belong to multiple categories, and a category can contain multiple resources.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `resource_id` | TEXT | `NOT NULL REFERENCES resources(id) ON DELETE CASCADE` | Resource BLAKE3 hash |
| `category_name` | TEXT | `NOT NULL REFERENCES categories(name) ON DELETE CASCADE` | Category name |

**Primary Key:** Composite `(resource_id, category_name)` — each resource appears at most once per category.
**Indexes:** Implicit primary key index on `(resource_id, category_name)`.

**Foreign Keys:**
- `resource_id` references `resources(id) ON DELETE CASCADE` — deleting a resource removes all its category assignments.
- `category_name` references `categories(name) ON DELETE CASCADE` — deleting a category removes all its resource mappings.

---

### 8. `share_resources`

Stores password-protected share links for individual resources. Each resource can have multiple passwords, enabling different share links with different access credentials for the same resource.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `resource_id` | TEXT | `NOT NULL REFERENCES resources(id) ON DELETE CASCADE` | Shared resource ID |
| `password` | TEXT | `NOT NULL` | Share password (stored as bcrypt hash) |
| `expires_at` | DATETINE | — | Optional expiration timestamp (NULL = never expires) |
| `created_by` | TEXT | `NOT NULL REFERENCES users(name) ON DELETE CASCADE` | User who created the share link |
| `created_at` | DATETIME | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Share creation timestamp |

**Primary Key:** Composite `(resource_id, password)` — each password per resource is unique.
**Indexes:** Implicit primary key index on `(resource_id, password)`.

**Foreign Keys:**
- `resource_id` references `resources(id) ON DELETE CASCADE` — deleting a resource removes all its share links.
- `created_by` references `users(name) ON DELETE CASCADE` — deleting a user removes all their created share links.

---

### 9. `playlists`

Defines ordered playlists within a category. A playlist belongs to exactly one category. Playlist names are unique within a category (enforced by the composite primary key).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `name` | TEXT | `NOT NULL` | Playlist identifier (URL-safe, unique per category) |
| `category_name` | TEXT | `NOT NULL REFERENCES categories(name) ON DELETE CASCADE` | Parent category |
| `playlist_type` | TEXT | `NOT NULL` | Playlist type discriminator (e.g., `video`, `audio`) |
| `display_name` | TEXT | `NOT NULL DEFAULT ''` | Human-readable playlist name |
| `description` | TEXT | `NOT NULL DEFAULT ''` | Playlist description |
| `created_by` | TEXT | `NOT NULL REFERENCES users(name)` | Admin who created the playlist |
| `sort_order` | INTEGER | `NOT NULL DEFAULT 0` | Display ordering position within the category |
| `created_at` | DATETIME | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Creation timestamp |

**Primary Key:** Composite `(category_name, name)` — playlist names are unique within a category, but two different categories can have playlists with the same name.
**Indexes:** Implicit primary key index on `(category_name, name)`.

**Foreign Keys:**
- `category_name` references `categories(name) ON DELETE CASCADE` — deleting a category removes all its playlists.
- `created_by` references `users(name)`. No cascade — deleting a user preserves their created playlists.

**Relationships:** Referenced by `playlist_videos` via the composite foreign key `(playlist_category_name, playlist_name)`.

---

### 10. `playlist_videos`

Maps resources to playlists with ordering. A resource can appear in multiple playlists, and a playlist can contain multiple resources. The `sort_order` column controls display position within the playlist.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `playlist_category_name` | TEXT | `NOT NULL` | Parent category name (part of composite FK to playlists) |
| `playlist_name` | TEXT | `NOT NULL` | Playlist name (part of composite FK to playlists) |
| `resource_id` | TEXT | `NOT NULL REFERENCES resources(id) ON DELETE CASCADE` | Resource in the playlist |
| `sort_order` | INTEGER | `NOT NULL DEFAULT 0` | Position within the playlist |

**Primary Key:** Composite `(playlist_category_name, playlist_name, resource_id)` — each resource appears at most once per playlist.
**Indexes:** Implicit primary key index on `(playlist_category_name, playlist_name, resource_id)`.

**Foreign Keys:**
- `(playlist_category_name, playlist_name)` references `playlists(category_name, name) ON DELETE CASCADE` — deleting a playlist removes all its resource mappings.
- `resource_id` references `resources(id) ON DELETE CASCADE` — deleting a resource removes it from all playlists.

---

### 11. `share_links`

Stores password-protected share links for categories and playlists (collection-level sharing). Unlike `share_resources` which targets individual resources, this table supports sharing entire collections.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | TEXT | `PRIMARY KEY` | Unique share link identifier |
| `password` | TEXT | `NOT NULL` | Share password (stored as bcrypt hash) |
| `target_type` | TEXT | `NOT NULL` | Target type: `category` or `playlist` |
| `target_id` | TEXT | `NOT NULL` | Target identifier (category name or playlist name) |
| `expires_at` | DATETIME | — | Optional expiration timestamp (NULL = never expires) |
| `created_by` | TEXT | `NOT NULL REFERENCES users(name) ON DELETE CASCADE` | User who created the link |
| `created_at` | DATETIME | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Link creation timestamp |

**Indexes:** Implicit primary key index on `id`.

**Foreign Key:** `created_by` references `users(name) ON DELETE CASCADE` — deleting a user removes all their created share links.

**Note:** `target_type` and `target_id` are not enforced by a foreign key constraint. The application validates that the target exists at runtime before granting access.
