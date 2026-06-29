INSERT INTO batch (name,rel_path,status) VALUES ('dir','dir','finished');

INSERT INTO image_hash (hash) VALUES
('5lp7PE6ZTl-Hiii8c5cPJ-Y8J1R04Zpe5xvwNaI-3r4')
ON CONFLICT (hash) DO NOTHING;

INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score, deleted_at) VALUES
(1, (SELECT id FROM image_hash WHERE hash = '5lp7PE6ZTl-Hiii8c5cPJ-Y8J1R04Zpe5xvwNaI-3r4'),  'image3.jpg',   1.0, 0.0,  NULL)