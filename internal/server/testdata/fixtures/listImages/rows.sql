INSERT INTO batch (name,rel_path,status) VALUES ('kek','kek','finished');
INSERT INTO batch (name,rel_path,status) VALUES ('lol','lol','finished');

INSERT INTO image_hash (hash) VALUES
('kekeke1'),
('kekeke2'),
('kekeke3'),
('kekeke4'),
('kekeke5'),
('kekeke6'),
('kekeke7'),
('kekeke8'),
('kekeke9'),
('kekeke10'),
('kekeke11')
ON CONFLICT (hash) DO NOTHING;


INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score) VALUES
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke1'), 'lol.png', 1.0, 1.0),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke2'), 'lol2.png', 0.75, 0.75),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke3'), 'lol3.png', 0.5, 0.50),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke4'), 'lol4.png', 0.25, 0.25),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke5'), 'lol5.png', 0.0, 0.0),
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke6'), 'lol6.png', 1.0, 1.0),
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke7'), 'lol7.png', 0.75, 0.75),
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke8'), 'lol8.png', 0.5, 0.5),
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke9'), 'lol9.png', 0.25, 0.25),
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke10'), 'lol10.png', 0.0, 0.0);

INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score, deleted_at) VALUES
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke11'), 'lol11.png', 0.0, 0.0, '2026-04-12 18:30:00');
