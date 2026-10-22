UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ'
      AND t.name = 'СМ №2'
)
WHERE code IN ('SASM-5', 'SASM-6');

DELETE FROM territories
WHERE name IN ('СМ 5', 'СМ 6')
  AND work_group_id = (
    SELECT id
    FROM work_groups
    WHERE name = 'СМ'
);