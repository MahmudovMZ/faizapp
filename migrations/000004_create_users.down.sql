ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_status_check;

ALTER TABLE users
DROP COLUMN IF EXISTS status;

UPDATE users
SET role = 'unassigned'
WHERE role IS NULL;

ALTER TABLE users
    ALTER COLUMN role SET NOT NULL;