DELETE FROM sr_codes
WHERE code IN ('SAHO-1', 'SAHO-2', 'SAHO-3', 'SAHO-4');

UPDATE brand_categories
SET name = 'Микс Хорека'
WHERE name = 'Микс ХорекаДи';