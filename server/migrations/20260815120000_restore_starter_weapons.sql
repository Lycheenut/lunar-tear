-- +goose Up

-- The three 2-star weapons granted by quest scene 2 cannot be obtained again.
-- Stage only genuinely missing weapons so this repair is safe to run once on
-- accounts that still own some or all of the original grants.
CREATE TEMP TABLE missing_starter_weapons (
    user_id           INTEGER NOT NULL,
    user_weapon_uuid  TEXT    NOT NULL,
    weapon_id         INTEGER NOT NULL,
    acquisition_datetime INTEGER NOT NULL,
    PRIMARY KEY (user_id, weapon_id)
);

INSERT INTO missing_starter_weapons (
    user_id, user_weapon_uuid, weapon_id, acquisition_datetime
)
SELECT
    users.user_id,
    lower(
        hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-' ||
        hex(randomblob(2)) || '-' || hex(randomblob(2)) || '-' ||
        hex(randomblob(6))
    ),
    starter_weapons.weapon_id,
    users.game_start_datetime
FROM users
CROSS JOIN (
    SELECT 100001 AS weapon_id
    UNION ALL SELECT 100011
    UNION ALL SELECT 100021
) AS starter_weapons
WHERE NOT EXISTS (
    SELECT 1
    FROM user_weapons
    WHERE user_weapons.user_id = users.user_id
      AND user_weapons.weapon_id = starter_weapons.weapon_id
);

INSERT INTO user_weapons (
    user_id, user_weapon_uuid, weapon_id, level, exp, limit_break_count,
    is_protected, acquisition_datetime, latest_version
)
SELECT
    user_id, user_weapon_uuid, weapon_id, 1, 0, 0,
    0, acquisition_datetime, 0
FROM missing_starter_weapons;

INSERT INTO user_weapon_skills (user_id, user_weapon_uuid, slot_number, level)
SELECT user_id, user_weapon_uuid, 1, 1
FROM missing_starter_weapons
UNION ALL
SELECT user_id, user_weapon_uuid, 2, 1
FROM missing_starter_weapons;

INSERT INTO user_weapon_abilities (user_id, user_weapon_uuid, slot_number, level)
SELECT user_id, user_weapon_uuid, 1, 1
FROM missing_starter_weapons;

INSERT OR IGNORE INTO user_weapon_notes (
    user_id, weapon_id, max_level, max_limit_break_count,
    first_acquisition_datetime, latest_version
)
SELECT user_id, weapon_id, 1, 0, acquisition_datetime, acquisition_datetime
FROM missing_starter_weapons;

INSERT OR IGNORE INTO user_weapon_stories (
    user_id, weapon_id, released_max_story_index, latest_version
)
SELECT user_id, weapon_id, 1, acquisition_datetime
FROM missing_starter_weapons;

DROP TABLE missing_starter_weapons;

-- +goose Down
-- Data repair is intentionally irreversible: after migration, restored weapons
-- cannot be distinguished safely from weapons granted through normal gameplay.
