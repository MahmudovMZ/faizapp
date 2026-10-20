ALTER TABLE photos
DROP COLUMN IF EXISTS telegram_file_id;

ALTER TABLE photos
    ADD COLUMN IF NOT EXISTS sha256_hash TEXT NOT NULL;

CREATE INDEX IF NOT EXISTS idx_photos_sha256_hash
    ON photos(sha256_hash);