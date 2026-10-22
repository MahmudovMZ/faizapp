-- ОПТ
UPDATE territories
SET name = 'Оптовики Фаровон'
WHERE name = 'Опт Фаровон';


-- ХОРЕКА

INSERT INTO work_groups (name)
VALUES ('Хорека №2')
    ON CONFLICT (name) DO NOTHING;

INSERT INTO territories (name, work_group_id)
SELECT territory_name, wg.id
FROM (
         VALUES
             ('Хорека 3'),
             ('Хорека 4')
     ) AS data(territory_name)
         JOIN work_groups wg ON wg.name = 'Хорека №2'
    ON CONFLICT (name, work_group_id) DO NOTHING;

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'Хорека'
      AND t.name = 'Хорека 1'
)
WHERE code = 'SAHO-1';

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'Хорека №1'
      AND t.name = 'Хорека 2'
)
WHERE code = 'SAHO-2';

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'Хорека №2'
      AND t.name = 'Хорека 3'
)
WHERE code = 'SAHO-3';

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'Хорека №2'
      AND t.name = 'Хорека 4'
)
WHERE code = 'SAHO-4';

UPDATE work_groups
SET name = 'Хорека №1'
WHERE name = 'Хорека';


-- СМ

INSERT INTO work_groups (name)
VALUES ('СМ №2')
    ON CONFLICT (name) DO NOTHING;

UPDATE work_groups
SET name = 'СМ №1'
WHERE name = 'СМ';

UPDATE territories
SET name = 'СМ 2'
WHERE name = 'СМ №1'
  AND work_group_id = (
    SELECT id FROM work_groups WHERE name = 'СМ №1'
);

UPDATE territories
SET name = 'СМ 3'
WHERE name = 'СМ №2'
  AND work_group_id = (
    SELECT id FROM work_groups WHERE name = 'СМ №1'
);

INSERT INTO territories (name, work_group_id)
SELECT territory_name, wg.id
FROM (
         VALUES
             ('СМ 1'),
             ('СМ 5'),
             ('СМ 6')
     ) AS data(territory_name)
         JOIN work_groups wg ON wg.name = 'СМ №1'
    ON CONFLICT (name, work_group_id) DO NOTHING;

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ №1'
      AND t.name = 'СМ 1'
)
WHERE code = 'SASM-1';

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ №1'
      AND t.name = 'СМ 2'
)
WHERE code = 'SASM-2';

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ №1'
      AND t.name = 'СМ 3'
)
WHERE code = 'SASM-3';

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ №1'
      AND t.name = 'СМ 2'
)
WHERE code = 'SASM-4';

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ №1'
      AND t.name = 'СМ 5'
)
WHERE code = 'SASM-5';

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ №1'
      AND t.name = 'СМ 6'
)
WHERE code = 'SASM-6';