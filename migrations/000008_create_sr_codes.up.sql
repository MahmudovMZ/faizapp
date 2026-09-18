CREATE TABLE IF NOT EXISTS sr_codes (
                                        id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                        territory_id INTEGER NOT NULL,
                                        category_id INTEGER NOT NULL,
                                        code TEXT NOT NULL UNIQUE,

                                        CONSTRAINT fk_sr_codes_territory
                                        FOREIGN KEY (territory_id)
    REFERENCES territories(id)
    ON DELETE RESTRICT,

    CONSTRAINT fk_sr_codes_category
    FOREIGN KEY (category_id)
    REFERENCES brand_categories(id)
    ON DELETE RESTRICT
    );