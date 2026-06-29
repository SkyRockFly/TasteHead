-- +goose Up
-- +goose StatementBegin
ALTER TABLE training
  ADD CONSTRAINT tag_image_unique
        UNIQUE (image_id,tag_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE training
    DROP CONSTRAINT IF EXISTS
tag_image_unique
-- +goose StatementEnd
