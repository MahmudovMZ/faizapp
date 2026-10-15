ALTER TABLE work_groups
DROP CONSTRAINT IF EXISTS fk_work_groups_supervisor;

ALTER TABLE work_groups
DROP COLUMN IF EXISTS supervisor_id;