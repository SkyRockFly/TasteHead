-- +goose Up
-- +goose StatementBegin
CREATE TABLE tag (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    deleted_at TIMESTAMP
);

INSERT INTO tag (name)
SELECT DISTINCT training_tag
FROM training
WHERE training_tag IS NOT NULL;

ALTER TABLE training
    ADD COLUMN tag_id BIGINT;

UPDATE training t
SET tag_id = tg.id
FROM tag tg
WHERE t.training_tag = tg.name;

ALTER TABLE training
    ALTER COLUMN tag_id SET NOT NULL,
    ADD CONSTRAINT training_tag_fk
        FOREIGN KEY (tag_id) REFERENCES tag (id);

ALTER TABLE training
    DROP COLUMN training_tag;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE training
    ADD COLUMN training_tag TEXT;

UPDATE training t
SET training_tag = tg.name
FROM tag tg
WHERE t.tag_id = tg.id;

ALTER TABLE training
    ALTER COLUMN training_tag SET NOT NULL;

ALTER TABLE training
    DROP CONSTRAINT training_tag_fk,
    DROP COLUMN tag_id,
    DROP CONSTRAINT IF EXISTS
        training_tag_fk,
    DROP CONSTRAINT IF EXISTS
        tag_image_unique,
    ADD CONSTRAINT
        UNIQUE(image_id,training_tag);

DROP TABLE tag;
-- +goose StatementEnd
