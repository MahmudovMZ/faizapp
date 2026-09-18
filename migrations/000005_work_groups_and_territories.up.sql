CREATE TABLE IF NOT EXISTS work_groups (
                                           id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                           name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS territories (
                                           id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                           name TEXT NOT NULL,
                                           work_group_id INTEGER NOT NULL,
                                           supervisor_id UUID NULL,

                                           CONSTRAINT fk_territories_work_group
                                           FOREIGN KEY (work_group_id)
    REFERENCES work_groups(id)
    ON DELETE RESTRICT,

    CONSTRAINT fk_territories_supervisor
    FOREIGN KEY (supervisor_id)
    REFERENCES users(id)
    ON DELETE SET NULL,

    CONSTRAINT uq_territory_per_group
    UNIQUE (name, work_group_id)
    );

INSERT INTO work_groups (name)
VALUES
    ('Розница'),
    ('ОПТ'),
    ('СМ'),
    ('РРП'),
    ('ХорекаДи')
    ON CONFLICT (name) DO NOTHING;