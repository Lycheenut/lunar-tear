-- +goose Up
-- These companions use placeholder enhancement materials and are distributed
-- with level-50 m_companion_enhanced templates. Older grants lost that level.
UPDATE user_companions
SET level = 50,
    latest_version = MAX(latest_version + 1, CAST(strftime('%s', 'now') AS INTEGER) * 1000)
WHERE companion_id IN (49, 50, 51) AND level < 50;

-- +goose Down
-- Data repair is intentionally irreversible: do not downgrade owned companions.
