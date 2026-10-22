INSERT INTO territories (name, work_group_id)
SELECT territory_name, wg.id
FROM (
         VALUES ('СМ 5'), ('СМ 6')
     ) AS data(territory_name)
         JOIN work_groups wg ON wg.name = 'СМ'
    ON CONFLICT (name, work_group_id) DO NOTHING;

UPDATE sr_codes
SET territory_id = (
    SELECT t.id
    FROM territories t
             JOIN work_groups wg ON wg.id = t.work_group_id
    WHERE wg.name = 'СМ'
      AND t.name = CASE
                       WHEN sr_codes.code = 'SASM-5' THEN 'СМ 5'
                       WHEN sr_codes.code = 'SASM-6' THEN 'СМ 6'
        END
)
WHERE code IN ('SASM-5', 'SASM-6');
