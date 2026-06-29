INSERT INTO batch (name,rel_path,status) VALUES ('kek','kek','FINISHED');

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
('kekeke11'),
('kekeke12')
ON CONFLICT (hash) DO NOTHING;


INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score, deleted_at) VALUES
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke1'),  'lol.png',   1.0, 0.0,  NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke2'),  'lol2.png',  1.0, 0.0,  NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke3'),  'lol3.png',  1.0, 0.25, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke4'),  'lol4.png',  1.0, 0.25, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke5'),  'lol5.png',  1.0, 0.5,  NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke6'),  'lol6.png',  1.0, 0.5,  NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke7'),  'lol7.png',  1.0, 0.75, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke8'),  'lol8.png',  1.0, 0.75, NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke9'),  'lol9.png',  1.0, 1.0,  NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke10'), 'lol10.png', 1.0, 1.0,  NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke11'), 'lol11.png', 1.0, 1.0,  '2026-04-12 18:30:00'),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke12'), 'lol12.png', 1.0, 1.0,  NULL);

INSERT INTO tag (name) VALUES ('dataset1');
INSERT INTO tag (name) VALUES ('dataset2');

INSERT INTO training (image_id,tag_id) VALUES (1,1);
INSERT INTO training (image_id,tag_id) VALUES (2,1);
INSERT INTO training (image_id,tag_id) VALUES (3,1);
INSERT INTO training (image_id,tag_id) VALUES (4,1);
INSERT INTO training (image_id,tag_id) VALUES (5,1);
INSERT INTO training (image_id,tag_id) VALUES (6,1);
INSERT INTO training (image_id,tag_id) VALUES (7,1);
INSERT INTO training (image_id,tag_id) VALUES (8,1);
INSERT INTO training (image_id,tag_id) VALUES (9,1);
INSERT INTO training (image_id,tag_id) VALUES (10,1);