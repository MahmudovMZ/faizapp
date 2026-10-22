-- =========================
-- ОПТ
-- =========================

UPDATE territories
SET name = 'Опт Фаровон'
WHERE name = 'Оптовики Фаровон';


-- =========================
-- ХОРЕКА
-- =========================

UPDATE work_groups
SET name = 'Хорека'
WHERE name = 'Хорека №1';

-- Хорека 1 получает SAHO-1 и SAHO-2
UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'Хорека'
      AND t.name = 'Хорека 1'
)
WHERE code IN ('SAHO-1', 'SAHO-2');

-- Хорека 2 получает SAHO-3 и SAHO-4
UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'Хорека'
      AND t.name = 'Хорека 2'
)
WHERE code IN ('SAHO-3', 'SAHO-4');

-- Удаляем старые территории Хорека 3 и Хорека 4
DELETE FROM territories
WHERE name IN ('Хорека 3', 'Хорека 4')
  AND work_group_id = (
    SELECT id FROM work_groups WHERE name = 'Хорека №2'
);

-- Удаляем старую рабочую группу
DELETE FROM work_groups
WHERE name = 'Хорека №2';


-- =========================
-- СМ
-- =========================

-- Основная рабочая группа СМ
UPDATE work_groups
SET name = 'СМ'
WHERE name = 'СМ №1';

-- Существующая территория СМ 2 становится СМ №1
UPDATE territories
SET name = 'СМ №1'
WHERE name = 'СМ 2'
  AND work_group_id = (
    SELECT id
    FROM work_groups
    WHERE name = 'СМ'
);

-- Существующая территория СМ 3 становится СМ №2
UPDATE territories
SET name = 'СМ №2'
WHERE name = 'СМ 3'
  AND work_group_id = (
    SELECT id
    FROM work_groups
    WHERE name = 'СМ'
);

-- СМ №1 получает SASM-1 и SASM-4
UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ'
      AND t.name = 'СМ №1'
)
WHERE code IN ('SASM-1', 'SASM-4');

-- СМ №2 получает SASM-2, SASM-3, SASM-5 и SASM-6
UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ'
      AND t.name = 'СМ №2'
)
WHERE code IN ('SASM-2', 'SASM-3', 'SASM-5', 'SASM-6');

-- Удаляем все оставшиеся территории старой группы СМ №2
DELETE FROM territories
WHERE work_group_id = (
    SELECT id
    FROM work_groups
    WHERE name = 'СМ №2'
);

-- Теперь старая группа удаляется без нарушения внешнего ключа
DELETE FROM work_groups
WHERE name = 'СМ №2';