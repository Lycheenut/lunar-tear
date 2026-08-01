-- +goose Up
CREATE TABLE user_costume_level_bonus_release_statuses (
    user_id INTEGER NOT NULL,
    costume_id INTEGER NOT NULL,
    last_released_bonus_level INTEGER NOT NULL DEFAULT 0,
    confirmed_bonus_level INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, costume_id)
);

CREATE TABLE user_costume_lottery_effect_abilities (
    user_id INTEGER NOT NULL,
    user_costume_uuid TEXT NOT NULL,
    slot_number INTEGER NOT NULL,
    ability_id INTEGER NOT NULL,
    ability_level INTEGER NOT NULL,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, user_costume_uuid, slot_number)
);

CREATE TABLE user_costume_lottery_effect_status_ups (
    user_id INTEGER NOT NULL,
    user_costume_uuid TEXT NOT NULL,
    status_calculation_type INTEGER NOT NULL,
    hp INTEGER NOT NULL DEFAULT 0,
    attack INTEGER NOT NULL DEFAULT 0,
    vitality INTEGER NOT NULL DEFAULT 0,
    agility INTEGER NOT NULL DEFAULT 0,
    critical_ratio INTEGER NOT NULL DEFAULT 0,
    critical_attack INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, user_costume_uuid, status_calculation_type)
);

CREATE TABLE user_deck_limit_content_restricted (
    user_id INTEGER NOT NULL,
    deck_restricted_uuid TEXT NOT NULL,
    event_quest_chapter_id INTEGER NOT NULL,
    quest_id INTEGER NOT NULL,
    possession_type INTEGER NOT NULL,
    target_uuid TEXT NOT NULL,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, deck_restricted_uuid)
);

CREATE TABLE user_cage_ornament_accesses (
    user_id INTEGER NOT NULL,
    cage_ornament_id INTEGER NOT NULL,
    first_access_datetime INTEGER NOT NULL,
    latest_access_datetime INTEGER NOT NULL,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, cage_ornament_id)
);

-- +goose Down
DROP TABLE user_cage_ornament_accesses;
DROP TABLE user_deck_limit_content_restricted;
DROP TABLE user_costume_lottery_effect_status_ups;
DROP TABLE user_costume_lottery_effect_abilities;
DROP TABLE user_costume_level_bonus_release_statuses;
