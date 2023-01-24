-- +goose Up
-- +goose StatementBegin
CREATE TABLE db_media_items (
    id integer PRIMARY KEY AUTOINCREMENT,
    created_at text,
    title text,
    subtitle text,
    category text,
    is_active boolean,
    source text,
    image text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE db_media_items;
-- +goose StatementEnd
