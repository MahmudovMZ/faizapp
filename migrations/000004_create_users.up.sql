ALTER TABLE users
    ADD COLUMN status TEXT;

UPDATE users
SET status = 'approved'
WHERE status IS NULL;

ALTER TABLE users
    ALTER COLUMN status SET DEFAULT 'pending';

ALTER TABLE users
    ALTER COLUMN status SET NOT NULL;

ALTER TABLE users
    ADD CONSTRAINT users_status_check
        CHECK (status IN ('pending', 'approved', 'rejected'));

ALTER TABLE users
    ALTER COLUMN role DROP NOT NULL;