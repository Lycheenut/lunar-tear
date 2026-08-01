-- +goose Up
CREATE TABLE user_mechanism_state (
    user_id INTEGER PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    state_json TEXT NOT NULL DEFAULT '{}'
);

-- +goose Down
DROP TABLE IF EXISTS user_mechanism_state;
