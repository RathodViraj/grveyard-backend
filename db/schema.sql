CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('buyer', 'founder')),
    password_hash TEXT NOT NULL,
    profile_pic_url TEXT,
    uuid TEXT UNIQUE NOT NULL,
    verified_at TIMESTAMP NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_is_deleted ON users(is_deleted);

CREATE TABLE IF NOT EXISTS startups (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    logo_url TEXT,                -- image stored as string
    owner_uuid TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'failed', 'sold')) DEFAULT 'failed',
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_startups_owner_uuid
        FOREIGN KEY (owner_uuid)
        REFERENCES users(uuid)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_startups_owner_uuid ON startups(owner_uuid);
CREATE INDEX IF NOT EXISTS idx_startups_is_deleted ON startups(is_deleted);

CREATE TABLE IF NOT EXISTS assets (
    id SERIAL PRIMARY KEY,
    user_uuid TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    asset_type TEXT NOT NULL CHECK (
        asset_type IN ('research', 'codebase', 'domain', 'product', 'data', 'other')
    ),
    image_url TEXT,               -- image stored as string
    price NUMERIC(12,2),
    is_negotiable BOOLEAN NOT NULL DEFAULT TRUE,
    is_sold BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_assets_user
        FOREIGN KEY (user_uuid)
        REFERENCES users(uuid)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_assets_user_uuid ON assets(user_uuid);
CREATE INDEX IF NOT EXISTS idx_assets_is_sold ON assets(is_sold);
CREATE INDEX IF NOT EXISTS idx_assets_is_active ON assets(is_active);
CREATE INDEX IF NOT EXISTS idx_assets_is_deleted ON assets(is_deleted);


CREATE TABLE IF NOT EXISTS chats (
    id SERIAL PRIMARY KEY,
    buyer_id INT NOT NULL,
    startup_id INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_chats_buyer
        FOREIGN KEY (buyer_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_chats_startup
        FOREIGN KEY (startup_id)
        REFERENCES startups(id)
        ON DELETE CASCADE,

    CONSTRAINT unique_chat UNIQUE (buyer_id, startup_id)
);

CREATE INDEX IF NOT EXISTS idx_chats_buyer_id ON chats(buyer_id);
CREATE INDEX IF NOT EXISTS idx_chats_startup_id ON chats(startup_id);

CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    chat_id INT NOT NULL,
    sender_id INT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_messages_chat
        FOREIGN KEY (chat_id)
        REFERENCES chats(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_messages_sender
        FOREIGN KEY (sender_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

-- =========================
-- OPTIONAL: TRANSACTIONS (future, but useful)
-- =========================
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    asset_id INT NOT NULL,
    buyer_id INT NOT NULL,
    final_price NUMERIC(12,2),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_transactions_asset
        FOREIGN KEY (asset_id)
        REFERENCES assets(id),

    CONSTRAINT fk_transactions_buyer
        FOREIGN KEY (buyer_id)
        REFERENCES users(id)
);

-- =========================
-- OTP TABLE
-- =========================
CREATE TABLE IF NOT EXISTS otps (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    code TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_otps_email ON otps(email);
CREATE INDEX IF NOT EXISTS idx_otps_expires_at ON otps(expires_at);
