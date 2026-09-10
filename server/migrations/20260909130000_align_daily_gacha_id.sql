-- +goose Up
-- The client uses daily Gacha 50030; older server builds stored it as 50001.
-- Keep the most recent business day, and the largest draw count on that day.
CREATE TEMP TABLE daily_gacha_id_repair AS
SELECT * FROM (
    SELECT b.*, d.count AS day_key,
           ROW_NUMBER() OVER (
               PARTITION BY b.user_id
               ORDER BY COALESCE(d.count, 0) DESC, b.draw_count DESC, b.gacha_id DESC
           ) AS priority
    FROM user_gacha_banners b
    LEFT JOIN user_gacha_banner_box_drew_counts d
      ON d.user_id = b.user_id AND d.gacha_id = b.gacha_id AND d.box_item_id = -2
    WHERE b.gacha_id IN (50001, 50030)
      AND EXISTS (
          SELECT 1 FROM user_gacha_banners old
          WHERE old.user_id = b.user_id AND old.gacha_id = 50001
      )
) WHERE priority = 1;

INSERT OR REPLACE INTO user_gacha_banners
    (user_id, gacha_id, medal_count, step_number, loop_count, draw_count, box_number)
SELECT user_id, 50030, medal_count, step_number, loop_count, draw_count, box_number
FROM daily_gacha_id_repair;

INSERT OR REPLACE INTO user_gacha_banner_box_drew_counts
    (user_id, gacha_id, box_item_id, count)
SELECT user_id, 50030, -2, day_key
FROM daily_gacha_id_repair WHERE day_key IS NOT NULL;

DELETE FROM user_gacha_banner_box_drew_counts WHERE gacha_id = 50001;
DELETE FROM user_gacha_banners WHERE gacha_id = 50001;
DROP TABLE daily_gacha_id_repair;

-- +goose Down
-- Keep the corrected IDs and draw history when rolling back schema versions.
