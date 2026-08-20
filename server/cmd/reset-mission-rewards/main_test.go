package main

import (
	"context"
	"database/sql"
	"testing"

	"lunar-tear/server/internal/model"

	_ "modernc.org/sqlite"
)

func TestResetClaimedMissionRewardsOnlyChangesClaimedTargets(t *testing.T) {
	db := openResetTestDB(t)
	insertResetTestMission(t, db, 1, 3700, model.MissionProgressStatusTypeRewardReceived, 10)
	insertResetTestMission(t, db, 1, 3701, model.MissionProgressStatusTypeRewardReceived, 11)
	insertResetTestMission(t, db, 1, 3702, model.MissionProgressStatusTypeClear, 12)
	insertResetTestMission(t, db, 2, 3705, model.MissionProgressStatusTypeInProgress, 13)
	insertResetTestMission(t, db, 2, 3711, model.MissionProgressStatusTypeRewardReceived, 14)
	insertResetTestMission(t, db, 2, 3712, model.MissionProgressStatusTypeRewardReceived, 15)

	stats, err := resetClaimedMissionRewards(context.Background(), db, false, 999)
	if err != nil {
		t.Fatal(err)
	}
	if stats != (resetStats{MissionRecords: 2, AffectedUsers: 2}) {
		t.Fatalf("reset stats = %+v", stats)
	}

	assertResetTestMission(t, db, 1, 3700, model.MissionProgressStatusTypeRewardReceived, 10)
	assertResetTestMission(t, db, 1, 3701, model.MissionProgressStatusTypeClear, 999)
	assertResetTestMission(t, db, 1, 3702, model.MissionProgressStatusTypeClear, 12)
	assertResetTestMission(t, db, 2, 3705, model.MissionProgressStatusTypeInProgress, 13)
	assertResetTestMission(t, db, 2, 3711, model.MissionProgressStatusTypeClear, 999)
	assertResetTestMission(t, db, 2, 3712, model.MissionProgressStatusTypeRewardReceived, 15)

	stats, err = resetClaimedMissionRewards(context.Background(), db, false, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if stats != (resetStats{}) {
		t.Fatalf("second reset stats = %+v, want no changes", stats)
	}
}

func TestResetClaimedMissionRewardsDryRunDoesNotWrite(t *testing.T) {
	db := openResetTestDB(t)
	insertResetTestMission(t, db, 1, 3701, model.MissionProgressStatusTypeRewardReceived, 10)

	stats, err := resetClaimedMissionRewards(context.Background(), db, true, 999)
	if err != nil {
		t.Fatal(err)
	}
	if stats != (resetStats{MissionRecords: 1, AffectedUsers: 1}) {
		t.Fatalf("dry-run stats = %+v", stats)
	}
	assertResetTestMission(t, db, 1, 3701, model.MissionProgressStatusTypeRewardReceived, 10)
}

func openResetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE user_missions (
		user_id INTEGER NOT NULL,
		mission_id INTEGER NOT NULL,
		start_datetime INTEGER NOT NULL,
		progress_value INTEGER NOT NULL,
		mission_progress_status_type INTEGER NOT NULL,
		clear_datetime INTEGER NOT NULL,
		latest_version INTEGER NOT NULL,
		PRIMARY KEY (user_id, mission_id))`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func insertResetTestMission(t *testing.T, db *sql.DB, userID int64, missionID int32, status model.MissionProgressStatusType, latestVersion int64) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO user_missions
		(user_id, mission_id, start_datetime, progress_value, mission_progress_status_type, clear_datetime, latest_version)
		VALUES (?,?,100,50,?,200,?)`, userID, missionID, status, latestVersion)
	if err != nil {
		t.Fatal(err)
	}
}

func assertResetTestMission(t *testing.T, db *sql.DB, userID int64, missionID int32, wantStatus model.MissionProgressStatusType, wantLatestVersion int64) {
	t.Helper()
	var startDatetime int64
	var progressValue int32
	var status int32
	var clearDatetime int64
	var latestVersion int64
	err := db.QueryRow(`SELECT start_datetime, progress_value, mission_progress_status_type, clear_datetime, latest_version
		FROM user_missions WHERE user_id=? AND mission_id=?`, userID, missionID).
		Scan(&startDatetime, &progressValue, &status, &clearDatetime, &latestVersion)
	if err != nil {
		t.Fatal(err)
	}
	if startDatetime != 100 || progressValue != 50 || clearDatetime != 200 {
		t.Fatalf("mission %d/%d completion data changed: start=%d progress=%d clear=%d",
			userID, missionID, startDatetime, progressValue, clearDatetime)
	}
	if status != int32(wantStatus) || latestVersion != wantLatestVersion {
		t.Fatalf("mission %d/%d status/version = %d/%d, want %d/%d",
			userID, missionID, status, latestVersion, wantStatus, wantLatestVersion)
	}
}
