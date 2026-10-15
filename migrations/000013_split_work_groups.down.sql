-- Возвращаем коды Хорека в одну территорию
UPDATE sr_codes sc
SET territory_id = t.id
    FROM territories t
WHERE sc.code IN ('SAHO-1', 'SAHO-2', 'SAHO-3', 'SAHO-4')
  AND t.name = 'Хорека 1';

-- Возвращаем коды СМ в одну территорию
UPDATE sr_codes sc
SET territory_id = t.id
    FROM territories t
WHERE sc.code IN (
    'SASM-1',
    'SASM-2',
    'SASM-3',
    'SASM-4',
    'SASM-5',
    'SASM-6'
    )
  AND t.name = 'СМ 2';

-- Удаляем новые территории
DELETE FROM territories
WHERE name IN (
               'Хорека 2',
               'Хорека 3',
               'Хорека 4',
               'СМ 1',
               'СМ 3',
               'СМ 5',
               'СМ 6'
    );

-- Возвращаем старые названия территорий
UPDATE territories
SET name = 'ХорекаДи'
WHERE name = 'Хорека 1';

UPDATE territories
SET name = 'СМ'
WHERE name = 'СМ 2';

-- Удаляем новые группы
DELETE FROM work_groups
WHERE name IN ('Хорека №2', 'СМ №2');

-- Возвращаем старые названия групп
UPDATE work_groups
SET name = 'ХорекаДи'
WHERE name = 'Хорека №1';

UPDATE work_groups
SET name = 'СМ'
WHERE name = 'СМ №1';