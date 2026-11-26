ALTER TABLE territories
    ADD COLUMN IF NOT EXISTS late_after TIME NOT NULL DEFAULT '10:30';

UPDATE territories SET late_after = '10:00' WHERE name = 'Опт Фаровон';