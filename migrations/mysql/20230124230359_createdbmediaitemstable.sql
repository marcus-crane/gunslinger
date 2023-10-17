-- +goose Up
-- +goose StatementBegin
CREATE TABLE db_media_items (
    `id` integer PRIMARY KEY AUTO_INCREMENT, -- NOTE: AUTOINCREMENT in sqlite, AUTO_INCREMENT in mysql
    `created_at` TEXT,
    `title` TEXT,
    `subtitle` TEXT,
    `category` TEXT,
    `is_active` BOOLEAN,
    `source` TEXT,
    `image` TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE db_media_items;
-- +goose StatementEnd
