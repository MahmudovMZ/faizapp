INSERT INTO brand_categories (name, photos_per_unit)
VALUES ('Микс СМ', 2)
ON CONFLICT (name) DO NOTHING;

UPDATE sr_codes SET category_id = (SELECT id FROM brand_categories WHERE name = 'Микс СМ')
WHERE code IN ('SASM-1', 'SASM-2', 'SASM-3');