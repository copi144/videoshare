CREATE TABLE resources (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    filename TEXT NOT NULL DEFAULT '',
    file_size INTEGER NOT NULL DEFAULT 0,
    content_type TEXT NOT NULL DEFAULT 'video/mp4',
    views INTEGER NOT NULL DEFAULT 0,
    uploaded_by TEXT,
    category_id TEXT,
    transcode_status TEXT NOT NULL DEFAULT 'none',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry DATETIME NOT NULL
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    totp_secret TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'uploader',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE category_uploaders (
    category_id TEXT NOT NULL REFERENCES categories(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    PRIMARY KEY (category_id, user_id)
);

CREATE TABLE playlists (
    id TEXT PRIMARY KEY,
    category_id TEXT NOT NULL REFERENCES categories(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL REFERENCES users(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE playlist_videos (
    playlist_id TEXT NOT NULL REFERENCES playlists(id),
    resource_id TEXT NOT NULL REFERENCES resources(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (playlist_id, resource_id)
);
