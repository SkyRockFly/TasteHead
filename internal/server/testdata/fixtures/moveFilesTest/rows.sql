INSERT INTO batch (name,rel_path,status) VALUES ('dir1','dir1','finished');
INSERT INTO batch (name,rel_path,status) VALUES ('dir2','dir2','finished');

INSERT INTO image_hash (hash) VALUES
('kekeke1'),
('kekeke2'),
('kekeke3')
ON CONFLICT (hash) DO NOTHING;

INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score) VALUES
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke1'), 'image.jpg', 1.0, 1.0),
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke2'), 'image2.webp', 0.75, 0.75),
(2, (SELECT id FROM image_hash WHERE hash = 'kekeke3'), 'image3.jpg', 1.0, 1.0);