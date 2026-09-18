CREATE TABLE IF NOT EXISTS sr_assignments (
                                              id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                              user_id UUID NOT NULL,
                                              sr_code_id INTEGER NOT NULL,
                                              assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_sr_assignments_user
    FOREIGN KEY (user_id)
    REFERENCES users(id)
    ON DELETE RESTRICT,

    CONSTRAINT fk_sr_assignments_sr_code
    FOREIGN KEY (sr_code_id)
    REFERENCES sr_codes(id)
    ON DELETE RESTRICT,

    CONSTRAINT uq_sr_code_assignment
    UNIQUE (sr_code_id)
    );