-- 用户
CREATE TABLE users (
    name TEXT PRIMARY KEY,
    totp_secret TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 会话（session 存储）
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry DATETIME NOT NULL
);

-- API 令牌
CREATE TABLE api_tokens (
    token TEXT PRIMARY KEY,
    username TEXT NOT NULL REFERENCES users(name),
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 分类
CREATE TABLE categories (
    name TEXT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL REFERENCES users(name),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 分类成员（用户-分类 多对多，含上传权限）
CREATE TABLE category_users (
    category_name TEXT NOT NULL REFERENCES categories(name) ON DELETE CASCADE,
    name TEXT NOT NULL REFERENCES users(name) ON DELETE CASCADE,
    can_upload INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (category_name, name)
);

-- 媒体资源
CREATE TABLE resources (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    filename TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    content_type TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    views INTEGER NOT NULL DEFAULT 0,
    uploaded_by TEXT NOT NULL REFERENCES users(name),
    transcode_status TEXT NOT NULL DEFAULT 'none',
    banned INTEGER NOT NULL DEFAULT 0,
    no_transcode INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 资源-分类 多对多
CREATE TABLE resource_categories (
    resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    category_name TEXT NOT NULL REFERENCES categories(name) ON DELETE CASCADE,
    PRIMARY KEY (resource_id, category_name)
);

-- 资源分享链接（每个资源+密码唯一）
CREATE TABLE share_resources (
    resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    password TEXT NOT NULL,
    expires_at DATETIME,
    created_by TEXT NOT NULL REFERENCES users(name) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (resource_id, password)
);

-- 播放列表
CREATE TABLE playlists (
    name TEXT NOT NULL,
    category_name TEXT NOT NULL REFERENCES categories(name) ON DELETE CASCADE,
    playlist_type TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL REFERENCES users(name),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (category_name, name)
);

-- 播放列表-资源 多对多
CREATE TABLE playlist_videos (
    playlist_category_name TEXT NOT NULL,
    playlist_name TEXT NOT NULL,
    resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (playlist_category_name, playlist_name, resource_id),
    FOREIGN KEY (playlist_category_name, playlist_name) REFERENCES playlists(category_name, name) ON DELETE CASCADE
);

-- 分类/播放列表分享链接
CREATE TABLE share_links (
    id TEXT PRIMARY KEY,
    password TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    expires_at DATETIME,
    created_by TEXT NOT NULL REFERENCES users(name) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
