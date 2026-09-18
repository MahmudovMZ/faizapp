DELETE FROM territories
WHERE (name, work_group_id) IN (
    SELECT territory.name, work_group.id
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
);