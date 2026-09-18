CREATE TABLE IF NOT EXISTS brand_categories (
                                                id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                                name TEXT NOT NULL UNIQUE,
                                                photos_per_unit INTEGER NOT NULL CHECK (photos_per_unit > 0)
    );

INSERT INTO brand_categories (name, photos_per_unit)
VALUES
    ('Бренд Мултон', 4),
    ('Бренд Махеев', 4),
    ('Бренд Эрман/Кампина', 2)
    ON CONFLICT (name) DO NOTHING;