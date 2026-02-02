-- Users table for mod developers
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT,
    is_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Mods table
CREATE TABLE IF NOT EXISTS mods (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    author_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    author_name TEXT, -- Fallback for legacy data or external authors
    type TEXT,
    thumbnail_url TEXT,
    website_url TEXT,
    latest_version_id TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Mod versions table
CREATE TABLE IF NOT EXISTS mod_versions (
    mod_id TEXT REFERENCES mods(id) ON DELETE CASCADE,
    version_id TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (mod_id, version_id)
);

-- Mod files table
CREATE TABLE IF NOT EXISTS mod_files (
    id SERIAL PRIMARY KEY,
    mod_id TEXT,
    version_id TEXT,
    file_type TEXT,
    path TEXT,
    url TEXT,
    compatible_binary_types TEXT[],
    FOREIGN KEY (mod_id, version_id) REFERENCES mod_versions(mod_id, version_id) ON DELETE CASCADE
);

-- Mod dependencies table
CREATE TABLE IF NOT EXISTS mod_dependencies (
    id SERIAL PRIMARY KEY,
    mod_id TEXT,
    version_id TEXT,
    dependency_id TEXT,
    dependency_version TEXT,
    dependency_type TEXT,
    FOREIGN KEY (mod_id, version_id) REFERENCES mod_versions(mod_id, version_id) ON DELETE CASCADE
);

-- Game version compatibility table
CREATE TABLE IF NOT EXISTS mod_version_game_versions (
    mod_id TEXT,
    version_id TEXT,
    game_version TEXT,
    PRIMARY KEY (mod_id, version_id, game_version),
    FOREIGN KEY (mod_id, version_id) REFERENCES mod_versions(mod_id, version_id) ON DELETE CASCADE
);
