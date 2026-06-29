INSERT INTO batch (name,rel_path,status) VALUES ('kek','kek','finished');
INSERT INTO batch (name,rel_path,status,deleted_at) VALUES ('dir','dir','finished','2026-04-12 18:30:00');
INSERT INTO batch (name,rel_path,status,deleted_at) VALUES ('dir2','dir2','finished','2026-04-12 18:30:00');


INSERT INTO image_hash (hash) VALUES
('kekeke1'),
('kekeke2'),
('kekeke3')
ON CONFLICT (hash) DO NOTHING;


INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score) VALUES
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke1'), 'lol.png', 1.0, 1.0),
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke2'), 'lol2.png', 0.75, 0.75),
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke3'), 'lol3.png', 0.5, 0.50);


