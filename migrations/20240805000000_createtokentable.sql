-- +goose Up
-- +goose StatementBegin
CREATE TABLE tokens (
    id TEXT PRIMARY KEY,
    value TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE tokens;
-- +goose StatementEnd
