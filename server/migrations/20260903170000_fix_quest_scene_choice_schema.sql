-- +goose Up
ALTER TABLE user_quest_scene_choices RENAME TO user_quest_scene_choices_legacy;

CREATE TABLE user_quest_scene_choices (
    user_id                         INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    quest_scene_choice_grouping_id  INTEGER NOT NULL,
    quest_scene_choice_effect_id    INTEGER NOT NULL,
    latest_version                  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, quest_scene_choice_grouping_id)
);

WITH ranked_choices AS (
    SELECT user_id,
           1 AS quest_scene_choice_grouping_id,
           choice_number AS quest_scene_choice_effect_id,
           latest_version,
           ROW_NUMBER() OVER (
               PARTITION BY user_id
               ORDER BY latest_version DESC, choice_datetime DESC, rowid DESC
           ) AS rank
    FROM user_quest_scene_choices_legacy
    WHERE quest_scene_id IN (1113, 1713)
      AND quest_flow_type IN (1, 3, 4)
      AND choice_number IN (1, 2, 3)
)
INSERT INTO user_quest_scene_choices (
    user_id, quest_scene_choice_grouping_id, quest_scene_choice_effect_id, latest_version
)
SELECT user_id, quest_scene_choice_grouping_id, quest_scene_choice_effect_id, latest_version
FROM ranked_choices
WHERE rank = 1;

DROP TABLE user_quest_scene_choices_legacy;

ALTER TABLE user_quest_scene_choice_history RENAME TO user_quest_scene_choice_history_legacy;

CREATE TABLE user_quest_scene_choice_history (
    user_id                       INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    quest_scene_choice_effect_id  INTEGER NOT NULL,
    choice_datetime               INTEGER NOT NULL DEFAULT 0,
    latest_version                INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, quest_scene_choice_effect_id)
);

INSERT INTO user_quest_scene_choice_history (
    user_id, quest_scene_choice_effect_id, choice_datetime, latest_version
)
SELECT user_id, choice_number, MIN(choice_datetime), MAX(latest_version)
FROM user_quest_scene_choice_history_legacy
WHERE quest_scene_id IN (1113, 1713)
  AND quest_flow_type IN (1, 3, 4)
  AND choice_number IN (1, 2, 3)
GROUP BY user_id, choice_number;

DROP TABLE user_quest_scene_choice_history_legacy;

-- +goose Down
ALTER TABLE user_quest_scene_choices RENAME TO user_quest_scene_choices_current;

CREATE TABLE user_quest_scene_choices (
    user_id          INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    quest_scene_id   INTEGER NOT NULL,
    quest_flow_type  INTEGER NOT NULL,
    choice_number    INTEGER NOT NULL,
    choice_datetime  INTEGER NOT NULL DEFAULT 0,
    latest_version   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, quest_scene_id, quest_flow_type)
);

INSERT INTO user_quest_scene_choices (
    user_id, quest_scene_id, quest_flow_type, choice_number, choice_datetime, latest_version
)
SELECT user_id, 1113, 3, quest_scene_choice_effect_id, latest_version, latest_version
FROM user_quest_scene_choices_current;

DROP TABLE user_quest_scene_choices_current;

ALTER TABLE user_quest_scene_choice_history RENAME TO user_quest_scene_choice_history_current;

CREATE TABLE user_quest_scene_choice_history (
    user_id          INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    quest_scene_id   INTEGER NOT NULL,
    quest_flow_type  INTEGER NOT NULL,
    choice_number    INTEGER NOT NULL,
    choice_datetime  INTEGER NOT NULL DEFAULT 0,
    latest_version   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, quest_scene_id, quest_flow_type, choice_number)
);

INSERT INTO user_quest_scene_choice_history (
    user_id, quest_scene_id, quest_flow_type, choice_number, choice_datetime, latest_version
)
SELECT user_id, 1113, 3, quest_scene_choice_effect_id, choice_datetime, latest_version
FROM user_quest_scene_choice_history_current;

DROP TABLE user_quest_scene_choice_history_current;
