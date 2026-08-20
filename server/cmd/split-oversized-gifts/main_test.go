package main

import (
	"context"
	"database/sql"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"

	_ "modernc.org/sqlite"
)

func TestRepairOversizedGiftsSplitsOnlyUnreceivedStackablePossessions(t *testing.T) {
	db := openRepairTestDB(t)
	config := &masterdata.GameConfig{
		ConsumableItemIdForGold:            99,
		PossessionCountLimitMaterial:       3,
		PossessionCountLimitConsumableItem: 4,
		PossessionCountLimitMoney:          10,
	}
	insertRepairTestGift(t, db, 1, "material", 0, int32(model.PossessionTypeMaterial), 10, 7)
	insertRepairTestGift(t, db, 1, "consumable", 0, int32(model.PossessionTypeConsumableItem), 20, 4)
	insertRepairTestGift(t, db, 1, "gold", 0, int32(model.PossessionTypeConsumableItem), 99, 11)
	insertRepairTestGift(t, db, 1, "weapon", 0, int32(model.PossessionTypeWeapon), 30, 20)
	insertRepairTestGift(t, db, 1, "received", 1, int32(model.PossessionTypeMaterial), 40, 8)
	if _, err := db.Exec(`INSERT INTO user_notification (user_id, gift_not_receive_count) VALUES (1,4)`); err != nil {
		t.Fatal(err)
	}

	stats, err := repairOversizedGifts(context.Background(), db, config, false)
	if err != nil {
		t.Fatal(err)
	}
	if stats != (repairStats{OversizedRecords: 2, CreatedRecords: 3, AffectedUsers: 1}) {
		t.Fatalf("repair stats = %+v", stats)
	}
	assertRepairTestCounts(t, db, "material", []int32{3, 3, 1})
	assertRepairTestCounts(t, db, "consumable", []int32{4})
	assertRepairTestCounts(t, db, "gold", []int32{10, 1})
	assertRepairTestCounts(t, db, "weapon", []int32{20})
	assertRepairTestCounts(t, db, "received", []int32{8})

	var notificationCount int32
	if err := db.QueryRow(`SELECT gift_not_receive_count FROM user_notification WHERE user_id=1`).Scan(&notificationCount); err != nil {
		t.Fatal(err)
	}
	if notificationCount != 7 {
		t.Fatalf("gift notification count = %d, want 7", notificationCount)
	}
	var originalCount int32
	if err := db.QueryRow(`SELECT count FROM user_gifts WHERE user_gift_uuid='material'`).Scan(&originalCount); err != nil {
		t.Fatal(err)
	}
	if originalCount != 3 {
		t.Fatalf("original gift count = %d, want 3", originalCount)
	}

	stats, err = repairOversizedGifts(context.Background(), db, config, false)
	if err != nil {
		t.Fatal(err)
	}
	if stats != (repairStats{}) {
		t.Fatalf("second repair stats = %+v, want no changes", stats)
	}
}

func TestRepairOversizedGiftsDryRunDoesNotWrite(t *testing.T) {
	db := openRepairTestDB(t)
	config := &masterdata.GameConfig{
		PossessionCountLimitMaterial:       3,
		PossessionCountLimitConsumableItem: 4,
		PossessionCountLimitMoney:          10,
	}
	insertRepairTestGift(t, db, 1, "material", 0, int32(model.PossessionTypeMaterial), 10, 7)

	stats, err := repairOversizedGifts(context.Background(), db, config, true)
	if err != nil {
		t.Fatal(err)
	}
	if stats != (repairStats{OversizedRecords: 1, CreatedRecords: 2, AffectedUsers: 1}) {
		t.Fatalf("dry-run stats = %+v", stats)
	}
	assertRepairTestCounts(t, db, "material", []int32{7})
}

func openRepairTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	statements := []string{
		`CREATE TABLE user_gifts (
			user_id INTEGER NOT NULL, user_gift_uuid TEXT NOT NULL, is_received INTEGER NOT NULL,
			possession_type INTEGER NOT NULL, possession_id INTEGER NOT NULL, count INTEGER NOT NULL,
			grant_datetime INTEGER NOT NULL, description_gift_text_id INTEGER NOT NULL,
			equipment_data BLOB, expiration_datetime INTEGER, received_datetime INTEGER,
			PRIMARY KEY (user_id, user_gift_uuid))`,
		`CREATE TABLE user_notification (user_id INTEGER PRIMARY KEY, gift_not_receive_count INTEGER NOT NULL)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func insertRepairTestGift(t *testing.T, db *sql.DB, userID int64, giftUUID string, received, possessionType, possessionID, count int32) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO user_gifts
		(user_id, user_gift_uuid, is_received, possession_type, possession_id, count, grant_datetime,
		 description_gift_text_id, equipment_data, expiration_datetime)
		VALUES (?,?,?,?,?,?,100,200,x'0102',300)`, userID, giftUUID, received, possessionType, possessionID, count)
	if err != nil {
		t.Fatal(err)
	}
}

func assertRepairTestCounts(t *testing.T, db *sql.DB, giftUUID string, want []int32) {
	t.Helper()
	rows, err := db.Query(`SELECT count FROM user_gifts
		WHERE user_id=1 AND (user_gift_uuid=? OR (possession_type, possession_id)=(
			SELECT possession_type, possession_id FROM user_gifts WHERE user_id=1 AND user_gift_uuid=?))
		ORDER BY rowid`, giftUUID, giftUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []int32
	for rows.Next() {
		var count int32
		if err := rows.Scan(&count); err != nil {
			t.Fatal(err)
		}
		got = append(got, count)
	}
	if len(got) != len(want) {
		t.Fatalf("gift %s counts = %v, want %v", giftUUID, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("gift %s counts = %v, want %v", giftUUID, got, want)
		}
	}
}
