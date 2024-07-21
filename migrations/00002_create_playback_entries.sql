-- +goose Up
-- +goose StatementBegin
CREATE TABLE playback_entries (
    id integer PRIMARY KEY AUTOINCREMENT,
    media_id TEXT,
    category TEXT,
    created_at DATETIME,
    elapsed INTEGER,
    status TEXT,
    is_active BOOLEAN,
    updated_at DATETIME,
    source TEXT,
    FOREIGN KEY(media_id) REFERENCES media_items(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE playback_entries;
-- +goose StatementEnd
