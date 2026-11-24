UPDATE sr_codes
SET category_id = (SELECT id FROM brand_categories WHERE name = 'Микс Мултон/Махеев')
WHERE code IN ('SASM-1', 'SASM-2', 'SASM-3');

DELETE FROM brand_categories WHERE name = 'Микс СМ';
