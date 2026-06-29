-- +goose Up
-- +goose StatementBegin
ALTER TABLE download
DROP CONSTRAINT IF EXISTS download_batch_id_fkey;

ALTER TABLE download
ADD CONSTRAINT download_batch_id_fkey
FOREIGN KEY (batch_id)
REFERENCES batch(id)
ON DELETE CASCADE;

ALTER TABLE training
DROP CONSTRAINT IF EXISTS training_image_id_fkey;

ALTER TABLE training
ADD CONSTRAINT training_image_id_fkey
FOREIGN KEY (image_id)
REFERENCES download(id)
ON DELETE CASCADE;

ALTER TABLE training
DROP CONSTRAINT IF EXISTS training_tag_id_fkey;

ALTER TABLE training
ADD CONSTRAINT training_tag_id_fkey
FOREIGN KEY (tag_id)
REFERENCES tag(id)
ON DELETE CASCADE;

CREATE TABLE image_hash(
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    hash TEXT NOT NULL UNIQUE
);

ALTER TABLE download
ADD COLUMN IF NOT EXISTS hash_id BIGINT REFERENCES image_hash(id) ON DELETE RESTRICT;

INSERT INTO image_hash (hash)
SELECT DISTINCT hash
FROM download
WHERE hash IS NOT NULL AND hash <> ''
ON CONFLICT (hash) DO NOTHING;

UPDATE download d
SET hash_id = ih.id
FROM image_hash ih
WHERE ih.hash = d.hash AND d.hash_id IS NULL;

ALTER TABLE download
ALTER COLUMN hash_id SET NOT NULL;

ALTER TABLE download
ADD CONSTRAINT download_hash_id_key
UNIQUE (hash_id);

ALTER TABLE download
DROP COLUMN IF EXISTS hash;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE download
ADD COLUMN hash TEXT;

UPDATE download d
SET hash = ih.hash
FROM image_hash ih
WHERE ih.id = d.hash_id

ALTER TABLE download
ALTER COLUMN hash SET NOT NULL 

ALTER TABLE download
ADD CONSTRAINT download_hash_key
UNIQUE (hash);

ALTER TABLE download
DROP CONSTRAINT IF EXISTS download_hash_id_key;

ALTER TABLE download
DROP COLUMN IF EXISTS hash_id;

DROP TABLE IF EXISTS image_hash;


ALTER TABLE training
DROP CONSTRAINT IF EXISTS training_tag_id_fkey;

ALTER TABLE training
ADD CONSTRAINT training_tag_id_fkey
FOREIGN KEY (tag_id)
REFERENCES tag(id);


ALTER TABLE training
DROP CONSTRAINT IF EXISTS training_image_id_fkey;

ALTER TABLE training
ADD CONSTRAINT training_image_id_fkey
FOREIGN KEY (image_id)
REFERENCES download(id);


ALTER TABLE download
DROP CONSTRAINT IF EXISTS download_batch_id_fkey;

ALTER TABLE download
ADD CONSTRAINT download_batch_id_fkey
FOREIGN KEY (batch_id)
REFERENCES batch(id);
-- +goose StatementEnd
