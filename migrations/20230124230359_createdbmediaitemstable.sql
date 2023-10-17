-- +goose Up
-- +goose StatementBegin
CREATE TABLE db_media_items (
    id integer PRIMARY KEY AUTOINCREMENT,
    created_at TEXT,
    title TEXT,
    subtitle TEXT,
    category TEXT,
    is_active BOOLEAN,
    source TEXT,
    image TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE db_media_items;
-- +goose StatementEnd
