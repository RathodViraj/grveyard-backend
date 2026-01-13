CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('buyer', 'founder')),
    password_hash TEXT NOT NULL,
    profile_pic_url TEXT,
    uuid TEXT UNIQUE NOT NULL,
    verified_at TIMESTAMP NULL,
    last_active_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
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
    -- revenue NUMERIC(12,2) DEFAULT 0.00,
    -- profit NUMERIC(12,2) DEFAULT 0.00,
    -- priority SMALLINT NOT NULL DEFAULT 0,
    -- intrested_buyers INT NOT NULL DEFAULT 0,

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
    -- priority SMALLINT NOT NULL DEFAULT 0,
    -- interested_buyers INT NOT NULL DEFAULT 0,

    CONSTRAINT fk_assets_user
        FOREIGN KEY (user_uuid)
        REFERENCES users(uuid)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_assets_user_uuid ON assets(user_uuid);
CREATE INDEX IF NOT EXISTS idx_assets_is_sold ON assets(is_sold);
CREATE INDEX IF NOT EXISTS idx_assets_is_active ON assets(is_active);
CREATE INDEX IF NOT EXISTS idx_assets_is_deleted ON assets(is_deleted);

CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,

    sender_id INT NOT NULL,
    receiver_id INT NOT NULL,

    content TEXT NOT NULL,

    message_type SMALLINT NOT NULL DEFAULT 0,
    -- 0 = text
    -- 1 = image
    -- 2 = file
    -- 3 = system (optional)

    is_read BOOLEAN NOT NULL DEFAULT FALSE,

    messaged_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,

    CONSTRAINT fk_messages_sender
        FOREIGN KEY (sender_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_messages_receiver
        FOREIGN KEY (receiver_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT chk_not_self_message
        CHECK (sender_id <> receiver_id)
);

CREATE INDEX IF NOT EXISTS idx_messages_pair
ON messages (
    LEAST(sender_id, receiver_id),
    GREATEST(sender_id, receiver_id),
    messaged_at
);

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
