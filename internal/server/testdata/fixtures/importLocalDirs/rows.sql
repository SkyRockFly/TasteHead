INSERT INTO batch (name,rel_path,status) VALUES ('updateDir','updateDir','finished');
INSERT INTO batch (name,rel_path,status) VALUES ('entryExistWithoutDir','entryExistWithoutDir','finished');

INSERT INTO image_hash (hash) VALUES
('F7Et5MymzC2NgC0opRmqZqIOfZ0K3QFkqbnp04O_bhI'),
('lHqCclFbN95JpBXXA-pOhynlZk1sRvAGXIPHpCtyc5I')
ON CONFLICT (hash) DO NOTHING;

INSERT INTO download (batch_id,hash_id,rel_path,model_score,user_score) VALUES
(1,(SELECT id FROM image_hash WHERE hash = 'F7Et5MymzC2NgC0opRmqZqIOfZ0K3QFkqbnp04O_bhI'),'image.jpg',1.0,1.0); /* existing hash */
INSERT INTO download (batch_id,hash_id,rel_path,model_score,user_score) VALUES
(1,(SELECT id FROM image_hash WHERE hash = 'lHqCclFbN95JpBXXA-pOhynlZk1sRvAGXIPHpCtyc5I'),'image2.webp',1.0,1.0);