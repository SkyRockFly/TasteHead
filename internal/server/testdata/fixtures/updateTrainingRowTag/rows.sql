INSERT INTO batch (name,rel_path,status) VALUES ('kek','kek','FINISHED');

INSERT INTO image_hash (hash) VALUES
('kekeke'),
('kekelo'),
('kelolo'),
('kekekeke'),
('kekelo5'),
('kekelo6')
ON CONFLICT (hash) DO NOTHING;


INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score, deleted_at) VALUES
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke'),   'lol.png',  1.0, 1.0, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekelo'),   'lol2.png', 1.0, 1.0, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kelolo'),   'lol3.png', 1.0, 1.0, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekekeke'), 'lol4.png', 1.0, 1.0, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekelo5'),  'lol5.png', 1.0, 1.0, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekelo6'),  'lol6.png', 1.0, 1.0, '2026-04-12 18:30:00');


INSERT INTO tag (name) VALUES ('dataset1');
INSERT INTO tag (name) VALUES ('dataset2');
INSERT INTO tag (name,deleted_at) VALUES ('dataset3','2026-04-12 18:30:00');

INSERT INTO training (image_id,tag_id) VALUES (1,1);
INSERT INTO training (image_id,tag_id) VALUES (2,1);
INSERT INTO training (image_id,tag_id) VALUES (3,1);
INSERT INTO training (image_id,tag_id) VALUES (4,1);
INSERT INTO training (image_id,tag_id) VALUES (5,1);
INSERT INTO training (image_id,tag_id) VALUES (6,1);

