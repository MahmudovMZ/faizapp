-- Telegram album ID; NULL for a single photo
ALTER TABLE photos
    ADD COLUMN IF NOT EXISTS media_group_id TEXT;

-- Find all photos of one album quickly; single photos are not indexed
CREATE INDEX IF NOT EXISTS idx_photos_media_group_id
    ON photos (media_group_id)
    WHERE media_group_id IS NOT NULL;
