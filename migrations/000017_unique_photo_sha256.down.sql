DROP INDEX IF EXISTS uq_photos_sha256_hash;

CREATE INDEX IF NOT EXISTS idx_photos_sha256_hash
    ON photos (sha256_hash);