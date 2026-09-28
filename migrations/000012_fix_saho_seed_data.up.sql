UPDATE brand_categories
SET name = 'Микс ХорекаДи'
WHERE name = 'Микс Хорека';

INSERT INTO sr_codes (territory_id, category_id, code)
SELECT
    t.id,
    bc.id,
    v.code
FROM (
    VALUES
        ('ХорекаДи', 'SAHO-1'),
        ('ХорекаДи', 'SAHO-2'),
        ('ХорекаДи', 'SAHO-3'),
        ('ХорекаДи', 'SAHO-4')
) AS v(territory_name, code)
JOIN territories t
    ON t.name = v.territory_name
JOIN brand_categories bc
    ON bc.name = 'Микс ХорекаДи'
ON CONFLICT (code) DO NOTHING;