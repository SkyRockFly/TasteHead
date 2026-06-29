INSERT INTO batch (name,rel_path,status) VALUES ('batch','batch','FINISHED');

INSERT INTO image_hash (hash) VALUES
('F7Et5MymzC2NgC0opRmqZqIOfZ0K3QFkqbnp04O_bhI'),
('lHqCclFbN95JpBXXA-pOhynlZk1sRvAGXIPHpCtyc5I'),
('5lp7PE6ZTl-Hiii8c5cPJ-Y8J1R04Zpe5xvwNaI-3r4')
ON CONFLICT (hash) DO NOTHING;

INSERT INTO download (batch_id, hash_id, rel_path, model_score, user_score, deleted_at) VALUES
(1, (SELECT id FROM image_hash WHERE hash = 'F7Et5MymzC2NgC0opRmqZqIOfZ0K3QFkqbnp04O_bhI'),'lol.png',1.0,1.0,NULL),
(1, (SELECT id FROM image_hash WHERE hash = 'lHqCclFbN95JpBXXA-pOhynlZk1sRvAGXIPHpCtyc5I'),'lol2.png',1.0,1.0,NULL),
(1, (SELECT id FROM image_hash WHERE hash = '5lp7PE6ZTl-Hiii8c5cPJ-Y8J1R04Zpe5xvwNaI-3r4'),'lol3.png',1.0,1.0,NULL);


INSERT INTO tag (name) VALUES ('dataset1');
INSERT INTO tag (name) VALUES ('dataset2');

INSERT INTO training (image_id,tag_id) VALUES (1,1);
INSERT INTO training (image_id,tag_id) VALUES (2,1);
INSERT INTO training (image_id,tag_id) VALUES (3,1);