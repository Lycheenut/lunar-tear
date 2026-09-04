package main

import (
	"context"
	"database/sql"
	"testing"

	"lunar-tear/server/internal/model"

	_ "modernc.org/sqlite"
)

func TestAdjustEligiblePlayerMaterials(t *testing.T) {
	db := openAdjustmentTestDB(t)
	insertAdjustmentTestUser(t, db, 1)
	insertAdjustmentTestUser(t, db, 2)
	insertAdjustmentTestUser(t, db, 3)
	insertAdjustmentTestUser(t, db, 4)
	insertAdjustmentTestUser(t, db, 5)
	insertAdjustmentTestMission(t, db, 1, targetMissionID, model.MissionProgressStatusTypeRewardReceived)
	insertAdjustmentTestMission(t, db, 2, targetMissionID, model.MissionProgressStatusTypeRewardReceived)
	insertAdjustmentTestMission(t, db, 3, targetMissionID, model.MissionProgressStatusTypeClear)
	insertAdjustmentTestMission(t, db, 4, 3710, model.MissionProgressStatusTypeRewardReceived)
	insertAdjustmentTestMaterial(t, db, 1, darkFateStoneMaterialID, 700)
	insertAdjustmentTestMaterial(t, db, 1, supremeAdmirationID, 10)
	insertAdjustmentTestMaterial(t, db, 1, 999999, 12)
	insertAdjustmentTestMaterial(t, db, 2, darkFateStoneMaterialID, 100)
	insertAdjustmentTestMaterial(t, db, 3, darkFateStoneMaterialID, 500)
	insertAdjustmentTestMaterial(t, db, 4, supremeAdmirationID, 20)
	insertAdjustmentTestMaterial(t, db, 5, darkFateStoneMaterialID, 900)

	stats, err := adjustEligiblePlayerMaterials(context.Background(), db, true)
	if err != nil {
		t.Fatal(err)
	}
	want := adjustmentStats{
		Players:                      2,
		DarkFateStonesBefore:         800,
		DarkFateStonesAfter:          -460,
		SupremeAdmirationBefore:      10,
		SupremeAdmirationAfter:       178,
		NegativeDarkFateStonePlayers: 1,
	}
	if stats != want {
		t.Fatalf("adjustment stats = %+v, want %+v", stats, want)
	}

	assertAdjustmentTestMaterial(t, db, 1, darkFateStoneMaterialID, 70)
	assertAdjustmentTestMaterial(t, db, 2, darkFateStoneMaterialID, -530)
	assertAdjustmentTestMaterial(t, db, 1, supremeAdmirationID, 94)
	assertAdjustmentTestMaterial(t, db, 2, supremeAdmirationID, 84)
	assertAdjustmentTestMaterial(t, db, 1, 999999, 12)
	assertAdjustmentTestMaterial(t, db, 3, darkFateStoneMaterialID, 500)
	assertAdjustmentTestMaterialMissing(t, db, 3, supremeAdmirationID)
	assertAdjustmentTestMaterialMissing(t, db, 4, darkFateStoneMaterialID)
	assertAdjustmentTestMaterial(t, db, 4, supremeAdmirationID, 20)
	assertAdjustmentTestMaterial(t, db, 5, darkFateStoneMaterialID, 900)
	assertAdjustmentTestMaterialMissing(t, db, 5, supremeAdmirationID)
}

func TestAdjustEligiblePlayerMaterialsDryRunDoesNotWrite(t *testing.T) {
	db := openAdjustmentTestDB(t)
	insertAdjustmentTestUser(t, db, 1)
	insertAdjustmentTestMission(t, db, 1, targetMissionID, model.MissionProgressStatusTypeRewardReceived)
	insertAdjustmentTestMaterial(t, db, 1, darkFateStoneMaterialID, 100)

	stats, err := adjustEligiblePlayerMaterials(context.Background(), db, false)
	if err != nil {
		t.Fatal(err)
	}
	want := adjustmentStats{
		Players:                      1,
		DarkFateStonesBefore:         100,
		DarkFateStonesAfter:          -530,
		SupremeAdmirationBefore:      0,
		SupremeAdmirationAfter:       84,
		NegativeDarkFateStonePlayers: 1,
	}
	if stats != want {
		t.Fatalf("dry-run stats = %+v, want %+v", stats, want)
	}
	assertAdjustmentTestMaterial(t, db, 1, darkFateStoneMaterialID, 100)
	assertAdjustmentTestMaterialMissing(t, db, 1, supremeAdmirationID)
}

func TestAdjustEligiblePlayerMaterialsHandlesNoPlayers(t *testing.T) {
	db := openAdjustmentTestDB(t)
	stats, err := adjustEligiblePlayerMaterials(context.Background(), db, true)
	if err != nil {
		t.Fatal(err)
	}
	if stats != (adjustmentStats{}) {
		t.Fatalf("empty adjustment stats = %+v", stats)
	}
}

func openAdjustmentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE users (user_id INTEGER PRIMARY KEY);
		CREATE TABLE user_missions (
			user_id INTEGER NOT NULL REFERENCES users(user_id),
			mission_id INTEGER NOT NULL,
			mission_progress_status_type INTEGER NOT NULL,
			PRIMARY KEY (user_id, mission_id));
		CREATE TABLE user_materials (
			user_id INTEGER NOT NULL REFERENCES users(user_id),
			material_id INTEGER NOT NULL,
			count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, material_id));`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func insertAdjustmentTestMission(t *testing.T, db *sql.DB, userID, missionID int64, status model.MissionProgressStatusType) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO user_missions (user_id, mission_id, mission_progress_status_type)
		VALUES (?, ?, ?)`, userID, missionID, status); err != nil {
		t.Fatal(err)
	}
}

func insertAdjustmentTestUser(t *testing.T, db *sql.DB, userID int64) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO users (user_id) VALUES (?)`, userID); err != nil {
		t.Fatal(err)
	}
}

func insertAdjustmentTestMaterial(t *testing.T, db *sql.DB, userID, materialID, count int64) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO user_materials (user_id, material_id, count) VALUES (?, ?, ?)`,
		userID, materialID, count); err != nil {
		t.Fatal(err)
	}
}

func assertAdjustmentTestMaterial(t *testing.T, db *sql.DB, userID, materialID, want int64) {
	t.Helper()
	var got int64
	if err := db.QueryRow(`SELECT count FROM user_materials WHERE user_id=? AND material_id=?`,
		userID, materialID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("material %d/%d count = %d, want %d", userID, materialID, got, want)
	}
}

func assertAdjustmentTestMaterialMissing(t *testing.T, db *sql.DB, userID, materialID int64) {
	t.Helper()
	var count int64
	err := db.QueryRow(`SELECT count FROM user_materials WHERE user_id=? AND material_id=?`,
		userID, materialID).Scan(&count)
	if err != sql.ErrNoRows {
		t.Fatalf("material %d/%d query error = %v, want sql.ErrNoRows", userID, materialID, err)
	}
}
