INSERT INTO brand_categories (name, photos_per_unit)
VALUES
    ('Микс Мултон/Махеев', 4),
    ('Микс Хорека', 2)
    ON CONFLICT (name) DO NOTHING;