-- +goose Up
-- +goose StatementBegin
ALTER TABLE playback_entries ADD COLUMN state_changed_at DATETIME;
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE playback_entries SET state_changed_at = updated_at WHERE state_changed_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE playback_entries DROP COLUMN state_changed_at;
-- +goose StatementEnd
