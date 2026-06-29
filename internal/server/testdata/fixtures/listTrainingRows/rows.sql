INSERT INTO batch (name,rel_path,status) VALUES ('kek','kek','finished');

INSERT INTO tag (name) VALUES ('dataset1');
INSERT INTO tag (name) VALUES ('dataset2');

INSERT INTO image_hash (hash) VALUES
('kekeke1'),
('kekeke2'),
('kekeke3'),
('kekeke4'),
('kekeke5')
ON CONFLICT (hash) DO NOTHING;

INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score) VALUES
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke1'), 'lol.png', 1.0, 1.0),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke2'), 'lol2.png', 0.75, 0.75),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke3'), 'lol3.png', 1.0, 1.0),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke4'), 'lol4.png', 0.25, 0.25),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke5'), 'lol5.png', 0.0, 0.0);

INSERT INTO training(image_id,tag_id) VALUES (1,1);
INSERT INTO training(image_id,tag_id) VALUES (2,1);
INSERT INTO training(image_id,tag_id) VALUES (3,2);
INSERT INTO training(image_id,tag_id) VALUES (4,2);
INSERT INTO training(image_id,tag_id,deleted_at) VALUES (5,1,'2026-04-12 18:30:00');

