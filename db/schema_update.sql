-- ALTER TABLE users
-- ADD COLUMN last_active_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT;

-- DROP TABLE IF EXISTS chats CASCADE;
-- DROP TABLE IF EXISTS messages CASCADE;

-- CREATE TABLE IF NOT EXISTS messages (
--     id BIGSERIAL PRIMARY KEY,

--     sender_id INT NOT NULL,
--     receiver_id INT NOT NULL,

--     content TEXT NOT NULL,

--     message_type SMALLINT NOT NULL DEFAULT 0,
--     -- 0 = text
--     -- 1 = image
--     -- 2 = file
--     -- 3 = system (optional)

--     is_read BOOLEAN NOT NULL DEFAULT FALSE,

--     messaged_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,

--     CONSTRAINT fk_messages_sender
--         FOREIGN KEY (sender_id)
--         REFERENCES users(id)
--         ON DELETE CASCADE,

--     CONSTRAINT fk_messages_receiver
--         FOREIGN KEY (receiver_id)
--         REFERENCES users(id)
--         ON DELETE CASCADE,

--     CONSTRAINT chk_not_self_message
--         CHECK (sender_id <> receiver_id)
-- );

-- CREATE INDEX IF NOT EXISTS idx_messages_pair
-- ON messages (
--     LEAST(sender_id, receiver_id),
--     GREATEST(sender_id, receiver_id),
--     messaged_at
-- );

