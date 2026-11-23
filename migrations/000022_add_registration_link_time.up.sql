ALTER TABLE users
    ADD COLUMN IF NOT EXISTS registration_link_sent_at TIMESTAMPTZ;