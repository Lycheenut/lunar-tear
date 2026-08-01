-- +goose Up
ALTER TABLE user_profile ADD COLUMN current_pvp_rank INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_profile ADD COLUMN current_pvp_grade_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_profile ADD COLUMN max_pvp_season_rank INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE user_profile DROP COLUMN max_pvp_season_rank;
ALTER TABLE user_profile DROP COLUMN current_pvp_grade_id;
ALTER TABLE user_profile DROP COLUMN current_pvp_rank;
