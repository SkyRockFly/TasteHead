INSERT INTO batch (name,rel_path,status) VALUES ('kek','kek','FINISHED');

INSERT INTO image_hash (hash) VALUES
('kekeke');
INSERT INTO image_hash (hash) VALUES
('kekelo');
INSERT INTO image_hash (hash) VALUES
('kelolo');
INSERT INTO image_hash (hash) VALUES
('kekekeke');
INSERT INTO image_hash (hash) VALUES
('kelolo5');
INSERT INTO image_hash (hash) VALUES
('kelolo6');

INSERT INTO download (batch_id,hash_id,rel_path,model_score,user_score) VALUES
(1,1,'lol.png',1.0,1.0);
INSERT INTO download (batch_id,hash_id,rel_path,model_score,user_score) VALUES
(1,2,'lol2.png',1.0,1.0);
INSERT INTO download (batch_id,hash_id,rel_path,model_score,user_score) VALUES
(1,3,'lol3.png',1.0,1.0);
INSERT INTO download (batch_id,hash_id,rel_path,model_score,user_score) VALUES
(1,4,'lol4.png',1.0,1.0);
INSERT INTO download (batch_id,hash_id,rel_path,model_score,user_score) VALUES
(1,5,'lol5.png',1.0,1.0);
INSERT INTO download (batch_id,hash_id,rel_path,model_score,user_score,deleted_at) VALUES
(1,6,'lol6.png',1.0,1.0,'2026-04-12 18:30:00');


INSERT INTO tag (name) VALUES ('dataset1');

INSERT INTO training (image_id,tag_id) VALUES (1,1);
INSERT INTO training (image_id,tag_id) VALUES (2,1);
INSERT INTO training (image_id,tag_id) VALUES (3,1);