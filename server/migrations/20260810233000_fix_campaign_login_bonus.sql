-- +goose Up
ALTER TABLE user_login_bonus RENAME TO user_login_bonus_legacy;

CREATE TABLE user_login_bonus (
    user_id                         INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    login_bonus_id                  INTEGER NOT NULL,
    current_page_number             INTEGER NOT NULL DEFAULT 0,
    current_stamp_number            INTEGER NOT NULL DEFAULT 0,
    latest_reward_receive_datetime  INTEGER NOT NULL DEFAULT 0,
    latest_version                  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, login_bonus_id)
);

INSERT INTO user_login_bonus (
    user_id, login_bonus_id, current_page_number, current_stamp_number,
    latest_reward_receive_datetime, latest_version
)
SELECT user_id, login_bonus_id, current_page_number, current_stamp_number,
       latest_reward_receive_datetime, latest_version
FROM user_login_bonus_legacy;

DROP TABLE user_login_bonus_legacy;

CREATE TABLE user_beginner_campaign (
    user_id                     INTEGER PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    beginner_campaign_id        INTEGER NOT NULL,
    campaign_register_datetime  INTEGER NOT NULL,
    latest_version              INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE user_comeback_campaign (
    user_id               INTEGER PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    comeback_campaign_id  INTEGER NOT NULL,
    comeback_datetime     INTEGER NOT NULL,
    latest_version        INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS user_comeback_campaign;
DROP TABLE IF EXISTS user_beginner_campaign;

ALTER TABLE user_login_bonus RENAME TO user_login_bonus_multi;

CREATE TABLE user_login_bonus (
    user_id                         INTEGER PRIMARY KEY REFERENCES users(user_id),
    login_bonus_id                  INTEGER NOT NULL DEFAULT 0,
    current_page_number             INTEGER NOT NULL DEFAULT 0,
    current_stamp_number            INTEGER NOT NULL DEFAULT 0,
    latest_reward_receive_datetime  INTEGER NOT NULL DEFAULT 0,
    latest_version                  INTEGER NOT NULL DEFAULT 0
);

INSERT INTO user_login_bonus (
    user_id, login_bonus_id, current_page_number, current_stamp_number,
    latest_reward_receive_datetime, latest_version
)
SELECT user_id, login_bonus_id, current_page_number, current_stamp_number,
       latest_reward_receive_datetime, latest_version
FROM user_login_bonus_multi AS candidate
WHERE login_bonus_id = (
    SELECT MIN(login_bonus_id)
    FROM user_login_bonus_multi
    WHERE user_id = candidate.user_id
);

DROP TABLE user_login_bonus_multi;
