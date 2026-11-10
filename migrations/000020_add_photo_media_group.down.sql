DROP INDEX IF EXISTS idx_photos_media_group_id;

ALTER TABLE photos
    DROP COLUMN IF EXISTS media_group_id;
