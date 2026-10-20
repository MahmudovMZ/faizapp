CREATE TABLE IF NOT EXISTS photos (
                                      id BIGSERIAL PRIMARY KEY,

                                      user_id UUID NOT NULL,
                                      telegram_user_id BIGINT NOT NULL,
                                      telegram_message_id INTEGER NOT NULL,
                                      telegram_file_id TEXT NOT NULL,

                                      caption TEXT NOT NULL DEFAULT '',

                                      received_at TIMESTAMPTZ NOT NULL,
                                      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_photos_user
    FOREIGN KEY (user_id)
    REFERENCES users(id)
    ON DELETE RESTRICT
    );

CREATE INDEX IF NOT EXISTS idx_photos_user_id
    ON photos(user_id);

CREATE INDEX IF NOT EXISTS idx_photos_received_at
    ON photos(received_at);

CREATE INDEX IF NOT EXISTS idx_photos_telegram_user_id
    ON photos(telegram_user_id);