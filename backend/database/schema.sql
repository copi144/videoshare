CREATE TABLE resources (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    filename TEXT NOT NULL DEFAULT '',
    file_size INTEGER NOT NULL DEFAULT 0,
    content_type TEXT NOT NULL DEFAULT 'video/mp4',
    resource_type TEXT NOT NULL DEFAULT 'video',
    views INTEGER NOT NULL DEFAULT 0,
    uploaded_by TEXT,
    category_name TEXT REFERENCES categories(name),
    transcode_status TEXT NOT NULL DEFAULT 'none',
    banned INTEGER NOT NULL DEFAULT 0,
    no_transcode INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry DATETIME NOT NULL
);

CREATE TABLE users (
    name TEXT PRIMARY KEY,
    totp_secret TEXT NOT NULL,
    display_name TEXT NOT NULL,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE categories (
    name TEXT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL REFERENCES users(name),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE category_users (
    category_name TEXT NOT NULL REFERENCES categories(name),
    name TEXT NOT NULL REFERENCES users(name),
    can_upload INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (category_name, name)
);

CREATE TABLE playlists (
    id TEXT PRIMARY KEY,
    category_name TEXT NOT NULL REFERENCES categories(name),
    playlist_type TEXT NOT NULL DEFAULT 'video',
    name TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL REFERENCES users(name),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE playlist_videos (
    playlist_id TEXT NOT NULL REFERENCES playlists(id),
    resource_id TEXT NOT NULL REFERENCES resources(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (playlist_id, resource_id)
);

CREATE TABLE api_tokens (
    token TEXT PRIMARY KEY,
    user_role TEXT NOT NULL,
    username TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Resource share links (one per resource + password combo)
CREATE TABLE share_resources (
    resource_id TEXT NOT NULL REFERENCES resources(id),
    password TEXT NOT NULL,
    expires_at DATETIME,
    created_by TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (resource_id, password)
);

-- Category / Playlist share links
CREATE TABLE share_links (
    id TEXT PRIMARY KEY,
    password TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    expires_at DATETIME,
    created_by TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
