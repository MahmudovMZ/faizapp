-- Queue of photos waiting to be processed.
-- One row per Telegram photo message; workers take rows from here.
CREATE TABLE IF NOT EXISTS photo_jobs (
                                          id                  BIGSERIAL PRIMARY KEY,

    -- what Telegram sent
                                          chat_id             BIGINT      NOT NULL,
                                          telegram_message_id INTEGER     NOT NULL,
                                          telegram_user_id    BIGINT      NOT NULL,
                                          sender_name         TEXT        NOT NULL DEFAULT '',
                                          file_id             TEXT        NOT NULL,
                                          caption             TEXT        NOT NULL DEFAULT '',
                                          media_group_id      TEXT,
                                          received_at         TIMESTAMPTZ NOT NULL,

    -- processing state
                                          status              TEXT        NOT NULL DEFAULT 'pending',
                                          attempts            INTEGER     NOT NULL DEFAULT 0,
                                          next_attempt_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_until        TIMESTAMPTZ,
    last_error          TEXT        NOT NULL DEFAULT '',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT photo_jobs_status_check
    CHECK (status IN ('pending', 'processing', 'done', 'dead')),

    -- the same Telegram message is queued only once
    CONSTRAINT uq_photo_jobs_message
    UNIQUE (chat_id, telegram_message_id)
    );

-- Workers look for jobs that are ready to run
CREATE INDEX IF NOT EXISTS idx_photo_jobs_ready
    ON photo_jobs (next_attempt_at)
    WHERE status IN ('pending', 'processing');