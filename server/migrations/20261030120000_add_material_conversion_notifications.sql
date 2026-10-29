-- +goose Up
-- Counts are only used to notify the client about new conversions. Existing
-- materials cannot tell us which were obtained from fragments; do not backfill.
CREATE TABLE user_material_conversions (
    user_id INTEGER NOT NULL REFERENCES users(user_id),
    material_id INTEGER NOT NULL,
    to_material_id INTEGER NOT NULL,
    converted_count INTEGER NOT NULL,
    convert_datetime INTEGER NOT NULL,
    PRIMARY KEY (user_id, material_id)
);

-- +goose Down
DROP TABLE user_material_conversions;
