    -- 1. Add the column, empty for now; numbers must not repeat
ALTER TABLE sr_codes
    ADD COLUMN IF NOT EXISTS position_in_report INTEGER UNIQUE;

-- 2. Fill the report order for every code
UPDATE sr_codes AS c
SET position_in_report = p.position_in_report
    FROM (VALUES
    ('SAM-1', 1),
    ('SAM-2', 2),
    ('SAM-3', 3),
    ('SAM-4', 4),
    ('SAM-5', 5),
    ('SAM-6', 6),
    ('SAM-7', 7),
    ('SAM-8', 8),
    ('SAE-1', 9),
    ('SAE-2', 10),
    ('SAE-3', 11),
    ('SAE-4', 12),
    ('SAE-5', 13),
    ('SAE-6', 14),
    ('SAE-7', 15),
    ('SAE-8', 16),
    ('SAMah-1', 17),
    ('SAMah-2', 18),
    ('SAMah-3', 19),
    ('SAMah-4', 20),
    ('SAMah-5', 21),
    ('SAMah-6', 22),
    ('SAMah-7', 23),
    ('SAMah-8', 24),
    ('SAHO-1', 25),
    ('SAHO-2', 26),
    ('SAHO-3', 27),
    ('SAHO-4', 28),
    ('SASM-1', 29),
    ('SASM-2', 30),
    ('SASM-3', 31),
    ('SASM-4', 32),
    ('SASM-5', 33),
    ('SASM-6', 34),
    ('SADV1', 35),
    ('SADV2', 36),
    ('SADV3', 37),
    ('SADT1', 38),
    ('SADT2', 39),
    ('SADT3', 40),
    ('SADR1', 41),
    ('SADR2', 42),
    ('SADR3', 43),
    ('SADG1', 44),
    ('SADG2', 45),
    ('SADG3', 46),
    ('SADF1', 47),
    ('SADF2', 48),
    ('SADFT1', 49),
    ('SADFT2', 50),
    ('SADFT3', 51)
) AS p(code, position_in_report)
WHERE c.code = p.code;

-- 3. From now on every code must have a position
ALTER TABLE sr_codes
    ALTER COLUMN position_in_report SET NOT NULL;