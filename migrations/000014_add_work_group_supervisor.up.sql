ALTER TABLE work_groups
    ADD COLUMN IF NOT EXISTS supervisor_id UUID NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_work_groups_supervisor'
    ) THEN
ALTER TABLE work_groups
    ADD CONSTRAINT fk_work_groups_supervisor
        FOREIGN KEY (supervisor_id)
            REFERENCES users(id)
            ON DELETE SET NULL;
END IF;
END
$$;