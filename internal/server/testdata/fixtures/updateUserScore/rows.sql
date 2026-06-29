INSERT INTO batch (name,rel_path,status) VALUES ('kek','kek','finished');

INSERT INTO image_hash (hash) VALUES
('kekeke1'),
('kekeke2'),
('kekeke3'),
('kekeke4'),
('kekeke5')
ON CONFLICT (hash) DO NOTHING;


INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score, deleted_at) VALUES
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke1'), 'lol.png',  1.0,  1.0,  NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke2'), 'lol2.png', 0.75, 0.75, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke3'), 'lol3.png', 0.5,  0.50, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke4'), 'lol4.png', 0.25, 0.25, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke5'), 'lol5.png', 0.0,  0.0,  '2026-04-12 18:30:00');