-- +goose Up
-- +goose StatementBegin
ALTER TABLE db_media_items ADD COLUMN duration_ms INTEGER NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- For reasons I don't really get yet, MySQL hates running migrations together

-- +goose StatementBegin
ALTER TABLE db_media_items ADD COLUMN dominant_colours TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE db_media_items DROP COLUMN duration_ms;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE db_media_items DROP COLUMN dominant_colours;
-- +goose StatementEnd