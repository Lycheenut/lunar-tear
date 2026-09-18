package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/migrations"
)

const repairTime int64 = 1800000000000

func testSchedule(t *testing.T) []stampReward {
	t.Helper()
	bonuses := []masterdata.EntityMLoginBonus{{LoginBonusId: 1, TotalPageCount: 10}}
	var stamps []masterdata.EntityMLoginBonusStamp
	for _, page := range []int32{1, 4, 7, 10} {
		for stamp := int32(1); stamp <= 15; stamp++ {
			tier := (page-1)/3 + 1
			row := masterdata.EntityMLoginBonusStamp{
				LoginBonusId: 1, LowerPageNumber: page, StampNumber: stamp,
				RewardPossessionType: 5, RewardPossessionId: 100000 + tier, RewardCount: 1,
			}
			if stamp == 2 {
				row.RewardPossessionType, row.RewardPossessionId, row.RewardCount = 6, 1, tier*1000
			}
			if stamp == 15 {
				row.RewardPossessionType, row.RewardPossessionId, row.RewardCount = 12, 0, 250+tier*50
			}
			stamps = append(stamps, row)
		}
	}
	schedule, err := makeSchedule(bonuses, stamps)
	if err != nil {
		t.Fatal(err)
	}
	return schedule
}

func fixture(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "game.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = openDatabase(path, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	execSQL(t, db, `
		INSERT INTO users (user_id,uuid,player_id) VALUES
		(1,'one',101),(2,'two',102),(3,'three',103),(4,'four',104),
		(5,'five',105),(6,'six',106),(7,'seven',107),(8,'eight',108);
		INSERT INTO user_login (user_id,total_login_count,last_login_datetime) VALUES
		(1,47,1799999999000),(2,1,1799740800000),(3,15,1799999999000),(4,0,0),
		(5,1,1799999999000),(6,2,1799999999000),(7,0,0),(8,1,1799999999000);
		INSERT INTO user_login_bonus (user_id,login_bonus_id,current_page_number,current_stamp_number,latest_reward_receive_datetime,latest_version) VALUES
		(1,1,3,14,1700000000000,10),(1,2,1,7,1700000000000,10),
		(3,1,1,15,1799999999000,10),(5,1,1,1,1799999999000,10),
		(6,1,0,0,0,10),(8,1,1,2,1799999999000,10);
		INSERT INTO user_materials (user_id,material_id,count) VALUES
		(1,315002,700),(3,315002,12),(4,315002,900),(5,315002,-5),(6,315002,20),(7,315002,-10),
		(1,888,999);
		INSERT INTO user_gem (user_id,paid_gem,free_gem) VALUES (1,88,17);
		INSERT INTO user_missions (user_id,mission_id,mission_progress_status_type,progress_value,start_datetime,clear_datetime,latest_version) VALUES
		(1,3711,9,40000,50,100,200),(1,3708,9,10000,50,100,200),
		(1,3709,2,20000,50,100,200),(1,3710,1,123,50,0,200),
		(2,3711,2,40000,50,100,200),(2,3710,9,30000,50,100,200),
		(3,3711,1,200,50,0,200),(3,3708,1,200,50,0,200),(3,3709,9,20000,50,100,200),
		(5,3711,0,0,50,0,200),(6,3711,9,40000,50,100,200),(7,3711,9,40000,50,100,200);
	`)
	return db, path
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func wantInt(t *testing.T, db *sql.DB, want int64, query string, args ...any) {
	t.Helper()
	var got int64
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("%s %v = %d, want %d", query, args, got, want)
	}
}

func TestRepairPreviewApplyAndRepeat(t *testing.T) {
	db, path := fixture(t)
	schedule := testSchedule(t)
	readOnly, err := openDatabase(path, false)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := repair(readOnly, schedule, false, repairTime)
	readOnly.Close()
	if err != nil {
		t.Fatal(err)
	}
	if preview.Applied || preview.ScannedPlayers != 8 || preview.StonePlayers != 4 || preview.ResetMissions != 3 || preview.LoginPlayers != 3 || preview.GrantedDays != 6 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if !reflect.DeepEqual(preview.AheadPlayers, []int64{8}) {
		t.Fatalf("ahead players: %v", preview.AheadPlayers)
	}
	wantInt(t, db, 700, `SELECT count FROM user_materials WHERE user_id=1 AND material_id=315002`)
	wantInt(t, db, 9, `SELECT mission_progress_status_type FROM user_missions WHERE user_id=1 AND mission_id=3708`)
	wantInt(t, db, 14, `SELECT current_stamp_number FROM user_login_bonus WHERE user_id=1 AND login_bonus_id=1`)
	wantInt(t, db, 0, `SELECT count(*) FROM sqlite_master WHERE name='player_data_repairs'`)

	applied, err := repair(db, schedule, true, repairTime)
	if err != nil {
		t.Fatal(err)
	}
	preview.Applied = true
	if !reflect.DeepEqual(applied, preview) {
		t.Fatal("apply report differs from preview")
	}
	for uid, want := range map[int]int64{1: 70, 2: -630, 3: 12, 4: 900, 5: -5, 6: -610, 7: -640} {
		wantInt(t, db, want, `SELECT count FROM user_materials WHERE user_id=? AND material_id=315002`, uid)
	}
	wantInt(t, db, 1, `SELECT count FROM user_materials WHERE user_id=1 AND material_id=100002`)
	wantInt(t, db, 999, `SELECT count FROM user_materials WHERE user_id=1 AND material_id=888`)
	wantInt(t, db, 2000, `SELECT count FROM user_consumable_items WHERE user_id=1 AND consumable_item_id=1`)
	wantInt(t, db, 317, `SELECT free_gem FROM user_gem WHERE user_id=1`)
	wantInt(t, db, 88, `SELECT paid_gem FROM user_gem WHERE user_id=1`)
	wantInt(t, db, 4, `SELECT current_page_number FROM user_login_bonus WHERE user_id=1 AND login_bonus_id=1`)
	wantInt(t, db, 2, `SELECT current_stamp_number FROM user_login_bonus WHERE user_id=1 AND login_bonus_id=1`)
	wantInt(t, db, 1799999999000, `SELECT latest_reward_receive_datetime FROM user_login_bonus WHERE user_id=1 AND login_bonus_id=1`)
	wantInt(t, db, 1799740800000, `SELECT latest_reward_receive_datetime FROM user_login_bonus WHERE user_id=2 AND login_bonus_id=1`)
	wantInt(t, db, 7, `SELECT current_stamp_number FROM user_login_bonus WHERE user_id=1 AND login_bonus_id=2`)
	wantInt(t, db, 10, `SELECT latest_version FROM user_login_bonus WHERE user_id=1 AND login_bonus_id=2`)
	wantInt(t, db, 2, `SELECT current_stamp_number FROM user_login_bonus WHERE user_id=8 AND login_bonus_id=1`)
	wantInt(t, db, 3, `SELECT count(*) FROM user_missions WHERE (user_id=1 AND mission_id=3708 OR user_id=2 AND mission_id=3710 OR user_id=3 AND mission_id=3709) AND mission_progress_status_type=2`)
	wantInt(t, db, 2, `SELECT mission_progress_status_type FROM user_missions WHERE user_id=1 AND mission_id=3709`)
	wantInt(t, db, 1, `SELECT mission_progress_status_type FROM user_missions WHERE user_id=1 AND mission_id=3710`)
	wantInt(t, db, 9, `SELECT mission_progress_status_type FROM user_missions WHERE user_id=1 AND mission_id=3711`)
	wantInt(t, db, 10000, `SELECT progress_value FROM user_missions WHERE user_id=1 AND mission_id=3708`)
	wantInt(t, db, 100, `SELECT clear_datetime FROM user_missions WHERE user_id=1 AND mission_id=3708`)
	wantInt(t, db, repairTime, `SELECT latest_version FROM user_missions WHERE user_id=1 AND mission_id=3708`)
	wantInt(t, db, repairTime, `SELECT latest_version FROM users WHERE user_id=1`)
	wantInt(t, db, 0, `SELECT latest_version FROM users WHERE user_id=4`)
	wantInt(t, db, 1, `SELECT count(*) FROM player_data_repairs`)

	// A repeat must not reset a reward claimed again after maintenance, or
	// compensate new login days/players introduced after the one-time repair.
	execSQL(t, db, `UPDATE user_missions SET mission_progress_status_type=9 WHERE user_id=1 AND mission_id=3708;
		UPDATE user_login SET total_login_count=48 WHERE user_id=1;
		INSERT INTO users (user_id,uuid) VALUES (9,'later');
		INSERT INTO user_missions (user_id,mission_id,mission_progress_status_type) VALUES (9,3711,9);`)
	repeated, err := repair(db, schedule, true, repairTime+1)
	if err != nil || !repeated.AlreadyApplied || repeated.Applied || len(repeated.Players) != 0 {
		t.Fatalf("repeat: %+v, %v", repeated, err)
	}
	wantInt(t, db, 70, `SELECT count FROM user_materials WHERE user_id=1 AND material_id=315002`)
	wantInt(t, db, 9, `SELECT mission_progress_status_type FROM user_missions WHERE user_id=1 AND mission_id=3708`)
	wantInt(t, db, 2, `SELECT current_stamp_number FROM user_login_bonus WHERE user_id=1 AND login_bonus_id=1`)
	wantInt(t, db, 0, `SELECT count(*) FROM user_materials WHERE user_id=9`)
}

func TestRepairRollsBackOnWriteFailure(t *testing.T) {
	db, _ := fixture(t)
	execSQL(t, db, `CREATE TRIGGER reject_repair BEFORE INSERT ON user_materials
		WHEN NEW.user_id=2 BEGIN SELECT RAISE(ABORT,'simulated write failure'); END`)
	if _, err := repair(db, testSchedule(t), true, repairTime); err == nil {
		t.Fatal("expected write failure")
	}
	wantInt(t, db, 700, `SELECT count FROM user_materials WHERE user_id=1 AND material_id=315002`)
	wantInt(t, db, 17, `SELECT free_gem FROM user_gem WHERE user_id=1`)
	wantInt(t, db, 9, `SELECT mission_progress_status_type FROM user_missions WHERE user_id=1 AND mission_id=3708`)
	wantInt(t, db, 14, `SELECT current_stamp_number FROM user_login_bonus WHERE user_id=1 AND login_bonus_id=1`)
	wantInt(t, db, 0, `SELECT count(*) FROM sqlite_master WHERE name='player_data_repairs'`)
}

func TestRepairRejectsInvalidData(t *testing.T) {
	for _, query := range []string{
		`UPDATE user_login_bonus SET current_stamp_number=16 WHERE user_id=1 AND login_bonus_id=1`,
		`UPDATE user_login SET total_login_count=151 WHERE user_id=2`,
		`UPDATE user_login SET last_login_datetime=0 WHERE user_id=2`,
		`UPDATE user_gem SET free_gem=2147483647 WHERE user_id=1`,
		`UPDATE user_materials SET count=-2147483648 WHERE user_id=1 AND material_id=315002`,
	} {
		t.Run(query, func(t *testing.T) {
			db, _ := fixture(t)
			execSQL(t, db, query)
			if _, err := repair(db, testSchedule(t), true, repairTime); err == nil {
				t.Fatal("expected invalid data error")
			}
			wantInt(t, db, 9, `SELECT mission_progress_status_type FROM user_missions WHERE user_id=1 AND mission_id=3708`)
			wantInt(t, db, 0, `SELECT count(*) FROM sqlite_master WHERE name='player_data_repairs'`)
		})
	}
}

func TestScheduleTemplatesAndValidation(t *testing.T) {
	schedule := testSchedule(t)
	for day, want := range map[int]stampReward{
		45: {3, 15, 12, 0, 300}, 46: {4, 1, 5, 100002, 1},
		90: {6, 15, 12, 0, 350}, 91: {7, 1, 5, 100003, 1},
		135: {9, 15, 12, 0, 400}, 136: {10, 1, 5, 100004, 1},
	} {
		if got := schedule[day-1]; got != want {
			t.Errorf("day %d = %+v, want %+v", day, got, want)
		}
	}
	bonuses := []masterdata.EntityMLoginBonus{{LoginBonusId: 1, TotalPageCount: 2}}
	valid := masterdata.EntityMLoginBonusStamp{LoginBonusId: 1, LowerPageNumber: 1, StampNumber: 1, RewardPossessionType: 5, RewardPossessionId: 1, RewardCount: 1}
	for _, kind := range []string{"missing", "duplicate", "unsupported", "gap"} {
		t.Run(kind, func(t *testing.T) {
			stamps := []masterdata.EntityMLoginBonusStamp{valid}
			switch kind {
			case "missing":
				stamps = nil
			case "duplicate":
				stamps = append(stamps, valid)
			case "unsupported":
				stamps[0].RewardPossessionType = 2
			case "gap":
				stamps[0].StampNumber = 2
			}
			if _, err := makeSchedule(bonuses, stamps); err == nil {
				t.Fatal("expected invalid schedule error")
			}
		})
	}
}

func TestOpenDatabaseDoesNotCreateMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	for _, apply := range []bool{false, true} {
		if db, err := openDatabase(path, apply); err == nil {
			db.Close()
			t.Fatal("opened missing database")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("missing database was created: %v", err)
		}
	}
}
