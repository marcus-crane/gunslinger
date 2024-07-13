-- +goose Up
-- +goose StatementBegin
CREATE TABLE playback_entries (
    id integer PRIMARY KEY AUTOINCREMENT,
    media_id TEXT,
    category TEXT,
    started_at DATETIME,
    elapsed INTEGER,
    status TEXT,
    is_active BOOLEAN,
    updated_at DATETIME,
    FOREIGN KEY(media_id) REFERENCES media_items(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE playback_entries;
-- +goose StatementEnd
