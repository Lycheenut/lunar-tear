-- +goose Up
CREATE TABLE user_battle_mission_details (
    user_id INTEGER PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    is_valid INTEGER NOT NULL DEFAULT 0,
    character_death_count INTEGER NOT NULL DEFAULT 0,
    max_damage INTEGER NOT NULL DEFAULT 0,
    costume_skill_use_count INTEGER NOT NULL DEFAULT 0,
    weapon_skill_use_count INTEGER NOT NULL DEFAULT 0,
    companion_skill_use_count INTEGER NOT NULL DEFAULT 0,
    critical_count INTEGER NOT NULL DEFAULT 0,
    combo_count INTEGER NOT NULL DEFAULT 0,
    combo_max_damage INTEGER NOT NULL DEFAULT 0,
    total_recover_point INTEGER NOT NULL DEFAULT 0,
    costume_result_count INTEGER NOT NULL DEFAULT 0,
    costume_1_is_alive INTEGER NOT NULL DEFAULT 0,
    costume_1_max_hp INTEGER NOT NULL DEFAULT 0,
    costume_1_remaining_hp INTEGER NOT NULL DEFAULT 0,
    costume_2_is_alive INTEGER NOT NULL DEFAULT 0,
    costume_2_max_hp INTEGER NOT NULL DEFAULT 0,
    costume_2_remaining_hp INTEGER NOT NULL DEFAULT 0,
    costume_3_is_alive INTEGER NOT NULL DEFAULT 0,
    costume_3_max_hp INTEGER NOT NULL DEFAULT 0,
    costume_3_remaining_hp INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE user_quest_replay_flow_rewards (
    user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    quest_replay_flow_reward_group_id INTEGER NOT NULL,
    reward_receive_datetime INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, quest_replay_flow_reward_group_id)
);

CREATE TABLE user_quest_scene_choices (
    user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    quest_scene_id INTEGER NOT NULL,
    quest_flow_type INTEGER NOT NULL,
    choice_number INTEGER NOT NULL,
    choice_datetime INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, quest_scene_id, quest_flow_type)
);

CREATE TABLE user_quest_scene_choice_history (
    user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    quest_scene_id INTEGER NOT NULL,
    quest_flow_type INTEGER NOT NULL,
    choice_number INTEGER NOT NULL,
    choice_datetime INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, quest_scene_id, quest_flow_type, choice_number)
);

CREATE TABLE user_event_quest_daily_rewards (
    user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    event_quest_daily_group_id INTEGER NOT NULL,
    reward_receive_datetime INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, event_quest_daily_group_id)
);

CREATE TABLE user_mission_pass_points (
    user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    mission_pass_id INTEGER NOT NULL,
    point INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, mission_pass_id)
);

CREATE TABLE user_mission_pass_rewards (
    user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    mission_pass_id INTEGER NOT NULL,
    level INTEGER NOT NULL,
    is_premium INTEGER NOT NULL,
    reward_receive_datetime INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, mission_pass_id, level, is_premium)
);

CREATE TABLE user_mission_pass_remaining (
    user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    mission_pass_id INTEGER NOT NULL,
    reward_received INTEGER NOT NULL DEFAULT 0,
    reward_receive_datetime INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, mission_pass_id)
);

CREATE TABLE user_webview_panel_missions (
    user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    webview_panel_mission_page_id INTEGER NOT NULL,
    reward_receive_datetime INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, webview_panel_mission_page_id)
);

-- +goose Down
DROP TABLE IF EXISTS user_webview_panel_missions;
DROP TABLE IF EXISTS user_mission_pass_remaining;
DROP TABLE IF EXISTS user_mission_pass_rewards;
DROP TABLE IF EXISTS user_mission_pass_points;
DROP TABLE IF EXISTS user_event_quest_daily_rewards;
DROP TABLE IF EXISTS user_quest_scene_choice_history;
DROP TABLE IF EXISTS user_quest_scene_choices;
DROP TABLE IF EXISTS user_quest_replay_flow_rewards;
DROP TABLE IF EXISTS user_battle_mission_details;
