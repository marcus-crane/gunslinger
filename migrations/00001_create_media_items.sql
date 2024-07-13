-- +goose Up
-- +goose StatementBegin
CREATE TABLE media_items (
    id TEXT PRIMARY KEY,
    title TEXT,
    subtitle TEXT,
    category TEXT,
    duration INTEGER,
    source TEXT,
    image TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE media_items;
-- +goose StatementEnd