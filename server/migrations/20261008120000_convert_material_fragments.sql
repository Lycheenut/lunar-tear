-- +goose Up
-- Repair fragments accumulated before automatic 10:1 conversion was implemented.
-- 501001: Piece: Zenith's Brilliance -> 322002: Zenith's Brilliance
-- 501002: Piece: Black Pearl -> 312003: Black Pearl
INSERT INTO user_materials (user_id, material_id, count)
SELECT user_id,
       CASE material_id WHEN 501001 THEN 322002 ELSE 312003 END,
       count / 10
FROM user_materials
WHERE material_id IN (501001, 501002) AND count >= 10
ON CONFLICT (user_id, material_id) DO UPDATE SET count = user_materials.count + excluded.count;

UPDATE user_materials
SET count = count % 10
WHERE material_id IN (501001, 501002) AND count >= 10;

DELETE FROM user_materials
WHERE material_id IN (501001, 501002) AND count = 0;

-- +goose Down
-- Converted materials may already have been consumed; do not recreate fragments.
