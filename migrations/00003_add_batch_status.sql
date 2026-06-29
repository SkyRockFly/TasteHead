-- +goose Up
-- +goose StatementBegin
ALTER TABLE batch
    ADD COLUMN status TEXT;

UPDATE batch b
SET status = 'finished';

ALTER TABLE batch
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN status SET DEFAULT 'pending';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE batch
    DROP COLUMN status;
-- +goose StatementEnd
