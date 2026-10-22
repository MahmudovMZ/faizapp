DROP INDEX IF EXISTS idx_photos_sha256_hash;

CREATE UNIQUE INDEX IF NOT EXISTS uq_photos_sha256_hash
    ON photos (sha256_hash);