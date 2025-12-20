-- Temporarily disable foreign key checks to allow data clearing
SET session_replication_role = replica;

-- Clear all data from tables being altered (safe deletion order to respect FKs)
DELETE FROM messages;
DELETE FROM chats;
DELETE FROM transactions;
DELETE FROM otps;
DELETE FROM assets;
DELETE FROM startups;
DELETE FROM users;

-- Re-enable foreign key checks
SET session_replication_role = default;

-- Ensure uuid is unique and not null for FK references
CREATE EXTENSION IF NOT EXISTS pgcrypto;
UPDATE users SET uuid = gen_random_uuid()::text WHERE uuid IS NULL OR uuid = '';
ALTER TABLE IF EXISTS users ALTER COLUMN uuid SET NOT NULL;
-- Use a unique index since ADD CONSTRAINT doesn't support IF NOT EXISTS
CREATE UNIQUE INDEX IF NOT EXISTS uniq_users_uuid ON users(uuid);
-- Migrations to shift owner_id -> owner_uuid and startup_id -> user_uuid

-- Startups: drop old FK/index, replace column
ALTER TABLE IF EXISTS startups DROP CONSTRAINT IF EXISTS fk_startups_owner;
DROP INDEX IF EXISTS idx_startups_owner_id;
ALTER TABLE IF EXISTS startups DROP COLUMN IF EXISTS owner_id;
ALTER TABLE IF EXISTS startups ADD COLUMN IF NOT EXISTS owner_uuid TEXT;
ALTER TABLE IF EXISTS startups ALTER COLUMN owner_uuid SET NOT NULL;
-- Only add FK if it doesn't already exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT constraint_name FROM information_schema.table_constraints 
        WHERE table_name = 'startups' AND constraint_name = 'fk_startups_owner_uuid'
    ) THEN
        ALTER TABLE startups ADD CONSTRAINT fk_startups_owner_uuid FOREIGN KEY (owner_uuid) REFERENCES users(uuid) ON DELETE CASCADE;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_startups_owner_uuid ON startups(owner_uuid);

-- Assets: drop old FK/index, replace column
ALTER TABLE IF EXISTS assets DROP CONSTRAINT IF EXISTS fk_assets_startup;
DROP INDEX IF EXISTS idx_assets_startup_id;
ALTER TABLE IF EXISTS assets DROP COLUMN IF EXISTS startup_id;
ALTER TABLE IF EXISTS assets ADD COLUMN IF NOT EXISTS user_uuid TEXT;
ALTER TABLE IF EXISTS assets ALTER COLUMN user_uuid SET NOT NULL;
-- Only add FK if it doesn't already exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT constraint_name FROM information_schema.table_constraints 
        WHERE table_name = 'assets' AND constraint_name = 'fk_assets_user'
    ) THEN
        ALTER TABLE assets ADD CONSTRAINT fk_assets_user FOREIGN KEY (user_uuid) REFERENCES users(uuid) ON DELETE CASCADE;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_assets_user_uuid ON assets(user_uuid);
