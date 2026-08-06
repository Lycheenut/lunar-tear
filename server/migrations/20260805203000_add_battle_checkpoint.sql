-- +goose Up
ALTER TABLE user_battle ADD COLUMN battle_binary BLOB;

-- +goose Down
ALTER TABLE user_battle DROP COLUMN battle_binary;
