-- +goose Up
-- +goose StatementBegin
ALTER TABLE db_media_items ADD COLUMN duration_ms INTEGER NOT NULL DEFAULT 1;
ALTER TABLE db_media_items ADD COLUMN dominant_colours TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE db_media_items DROP COLUMN duration_ms;
ALTER TABLE db_media_items DROP COLUMN dominant_colours;
-- +goose StatementEnd
