INSERT INTO batch (name,rel_path,status) VALUES ('kek','kek','finished');

INSERT INTO image_hash (hash) VALUES
('kekeke1'),
('kekeke2')
ON CONFLICT (hash) DO NOTHING;

INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score, deleted_at) VALUES
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke1'),  'lol.png',   1.0, 0.0,  NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke2'),  'lol2.png',  1.0, 0.0,  '2026-04-12 18:30:00');



