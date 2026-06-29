-- +goose Up
-- +goose StatementBegin
CREATE TABLE batch (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    rel_path TEXT NOT NULL UNIQUE,
    deleted_at TIMESTAMP
);

CREATE TABLE download ( 
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  batch_id BIGINT NOT NULL REFERENCES batch(id),
  hash TEXT NOT NULL UNIQUE,
  rel_path TEXT NOT NULL,
  model_score REAL NOT NULL,
  user_score real,
  deleted_at TIMESTAMP
);

CREATE TABLE training (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    image_id BIGINT NOT NULL REFERENCES download(id),
    training_tag TEXT NOT NULL,
    deleted_at TIMESTAMP,
    UNIQUE(image_id, training_tag)
);
-- +goose StatementEnd



-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
