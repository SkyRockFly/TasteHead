INSERT INTO batch (name,rel_path,status) VALUES ('dir','dir','finished');
INSERT INTO batch (name,rel_path,status,deleted_at) VALUES ('dir2','dir2','finished','2026-04-12 18:30:00');


INSERT INTO image_hash (hash) VALUES
('kekeke1'),
('kekeke2'),
('kekeke3'),
('kekeke4'),
('kekeke5')
ON CONFLICT (hash) DO NOTHING;


INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score,deleted_at) VALUES
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke1'), 'image.jpg', 1.0, 1.0,'2026-04-12 18:30:00'),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke2'), 'image2.webp', 0.75, 0.75,'2026-04-12 18:30:00'),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke3'), 'image3.jpg', 0.5, 0.50,'2026-04-12 18:30:00'),
(1, (SELECT id FROM image_hash WHERE hash = 'kekeke4'), 'image4.jpg', 0.5, 0.50,NULL);


