INSERT INTO territories (name, work_group_id)
SELECT territory.name, group_ref.id
FROM (
         VALUES
             ('Розница 1', 'Розница'),
             ('Розница 2', 'Розница'),
             ('Розница 3', 'Розница'),
             ('Розница 4', 'Розница'),
             ('Оптовики Фаровон', 'ОПТ'),
             ('СМ', 'СМ'),
             ('Вахдат', 'РРП'),
             ('Рудаки', 'РРП'),
             ('Гиссар', 'РРП'),
             ('Турсунзаде', 'РРП'),
             ('ХорекаДи', 'ХорекаДи')
     ) AS territory(name, group_name)
         JOIN work_groups AS group_ref
              ON group_ref.name = territory.group_name
    ON CONFLICT (name, work_group_id) DO NOTHING;