-- Переименовываем существующие группы
UPDATE work_groups
SET name = 'Хорека №1'
WHERE name = 'ХорекаДи';

UPDATE work_groups
SET name = 'СМ №1'
WHERE name = 'СМ';

-- Добавляем новые группы
INSERT INTO work_groups (name)
VALUES
    ('Хорека №2'),
    ('СМ №2')
    ON CONFLICT (name) DO NOTHING;

-- Переименовываем существующие территории
UPDATE territories
SET name = 'Хорека 1'
WHERE name = 'ХорекаДи';

UPDATE territories
SET name = 'СМ 2'
WHERE name = 'СМ';

-- Добавляем территории Хорека
INSERT INTO territories (name, work_group_id)
SELECT v.name, wg.id
FROM (
         VALUES
             ('Хорека 2', 'Хорека №1'),
             ('Хорека 3', 'Хорека №2'),
             ('Хорека 4', 'Хорека №2')
     ) AS v(name, group_name)
         JOIN work_groups wg ON wg.name = v.group_name
    ON CONFLICT (name, work_group_id) DO NOTHING;

-- Добавляем территории СМ
INSERT INTO territories (name, work_group_id)
SELECT v.name, wg.id
FROM (
         VALUES
             ('СМ 1', 'СМ №2'),
             ('СМ 3', 'СМ №1'),
             ('СМ 5', 'СМ №1'),
             ('СМ 6', 'СМ №1')
     ) AS v(name, group_name)
         JOIN work_groups wg ON wg.name = v.group_name
    ON CONFLICT (name, work_group_id) DO NOTHING;

-- Переносим коды Хорека
UPDATE sr_codes sc
SET territory_id = t.id
    FROM territories t
WHERE sc.code = 'SAHO-2'
  AND t.name = 'Хорека 2';

UPDATE sr_codes sc
SET territory_id = t.id
    FROM territories t
WHERE sc.code = 'SAHO-3'
  AND t.name = 'Хорека 3';

UPDATE sr_codes sc
SET territory_id = t.id
    FROM territories t
WHERE sc.code = 'SAHO-4'
  AND t.name = 'Хорека 4';

-- Переносим коды СМ
UPDATE sr_codes sc
SET territory_id = t.id
    FROM territories t
WHERE sc.code = 'SASM-1'
  AND t.name = 'СМ 1';

UPDATE sr_codes sc
SET territory_id = t.id
    FROM territories t
WHERE sc.code = 'SASM-3'
  AND t.name = 'СМ 3';

UPDATE sr_codes sc
SET territory_id = t.id
    FROM territories t
WHERE sc.code = 'SASM-5'
  AND t.name = 'СМ 5';

UPDATE sr_codes sc
SET territory_id = t.id
    FROM territories t
WHERE sc.code = 'SASM-6'
  AND t.name = 'СМ 6';