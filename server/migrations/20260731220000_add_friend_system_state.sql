-- +goose Up
CREATE TABLE user_friends (
    user_id                   INTEGER NOT NULL REFERENCES users(user_id),
    friend_user_id            INTEGER NOT NULL,
    is_friend                 INTEGER NOT NULL DEFAULT 1,
    cheer_sent_datetime       INTEGER NOT NULL DEFAULT 0,
    cheer_received_datetime   INTEGER NOT NULL DEFAULT 0,
    stamina_received_datetime INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, friend_user_id)
);

CREATE TABLE user_friend_requests (
    user_id            INTEGER NOT NULL REFERENCES users(user_id),
    requester_user_id  INTEGER NOT NULL,
    request_datetime   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, requester_user_id)
);

-- +goose Down
DROP TABLE IF EXISTS user_friend_requests;
DROP TABLE IF EXISTS user_friends;
