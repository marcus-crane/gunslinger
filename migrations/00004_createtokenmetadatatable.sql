-- +goose Up
-- +goose StatementBegin
CREATE TABLE tokenmetadata (
    id TEXT PRIMARY KEY,
    createdat INTEGER,
    expiresin INTEGER
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE tokenmetadata;
-- +goose StatementEnd
