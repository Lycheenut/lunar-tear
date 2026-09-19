package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func fixture(t *testing.T) (*sql.DB, string, []store.UserState) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "game.db")
	db, err := database.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	var users []store.UserState
	for _, phase := range []int32{19, 20, 99999} {
		id, err := repo.CreateUser("repair-"+strconv.Itoa(int(phase)), model.ClientPlatform{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = repo.UpdateUser(id, func(user *store.UserState) {
			user.Tutorials[3] = store.TutorialProgressState{TutorialType: 3, ProgressPhase: phase}
			user.Materials[315002] = 700
			user.Login.TotalLoginCount = 47
			user.LoginBonuses[1] = store.UserLoginBonusState{LoginBonusId: 1, CurrentPageNumber: 1, CurrentStampNumber: 2}
			user.Missions[3711] = store.UserMissionState{MissionId: 3711, MissionProgressStatusType: int32(model.MissionProgressStatusTypeRewardReceived)}
		})
		if err != nil {
			t.Fatal(err)
		}
		user, err := repo.LoadUser(id)
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, user)
	}
	return db, path, users
}

func TestRepairPreviewAndApply(t *testing.T) {
	masterPath := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	info, err := os.Stat(masterPath)
	if err != nil {
		t.Fatal(err)
	}
	granter, err := loadGranter(masterPath)
	if err != nil {
		t.Fatal(err)
	}
	currentInfo, err := os.Stat(masterPath)
	if err != nil || !currentInfo.ModTime().Equal(info.ModTime()) {
		t.Fatal("loading repair master data changed the source timestamp")
	}
	db, path, before := fixture(t)
	readOnly, err := openDatabase(path, false)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := repair(readOnly, granter, false, 1000)
	readOnly.Close()
	if err != nil {
		t.Fatal(err)
	}
	if preview.Applied || preview.ScannedPlayers != 3 || len(preview.Players) != 2 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	repo := sqlite.New(db, nil)
	for _, user := range before {
		after, err := repo.LoadUser(user.UserId)
		if err != nil || !reflect.DeepEqual(user, after) {
			t.Fatalf("preview changed user %d: %v", user.UserId, err)
		}
	}
	applied, err := repair(db, granter, true, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || !reflect.DeepEqual(preview.Players, applied.Players) {
		t.Fatalf("application does not match preview: %+v", applied)
	}
	for i, user := range before {
		after, err := repo.LoadUser(user.UserId)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if !reflect.DeepEqual(user, after) {
				t.Fatal("locked player was modified")
			}
			continue
		}
		if len(after.Costumes) != len(user.Costumes)+1 || len(after.Weapons) != len(user.Weapons)+1 {
			t.Fatal("expected one costume and one weapon")
		}
		for id, count := range map[int32]int32{311211: 40, 313197: 5, 312011: 4} {
			if after.Materials[id]-user.Materials[id] != count {
				t.Errorf("material %d: expected delta %d", id, count)
			}
		}
		if !reflect.DeepEqual(user.Tutorials, after.Tutorials) || !reflect.DeepEqual(user.Gifts, after.Gifts) ||
			!reflect.DeepEqual(user.LoginBonuses, after.LoginBonuses) || !reflect.DeepEqual(user.Login, after.Login) ||
			after.Materials[315002] != 700 || after.Missions[3711] != user.Missions[3711] {
			t.Fatal("backfill changed tutorials, gifts or data handled by the removed repair")
		}
	}
	var repairTables int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='player_data_repairs'`).Scan(&repairTables); err != nil || repairTables != 0 {
		t.Fatalf("repair marker table created: count=%d error=%v", repairTables, err)
	}
}

func TestRepairRollsBackAllPlayersOnWriteFailure(t *testing.T) {
	db, _, before := fixture(t)
	granter := &store.PossessionGranter{
		CostumeById: map[int32]store.CostumeRef{24008: {CharacterId: 24}},
		WeaponById:  map[int32]store.WeaponRef{240271: {}},
	}
	if _, err := db.Exec(`CREATE TRIGGER reject_backfill BEFORE INSERT ON user_materials
		WHEN NEW.user_id=3 AND NEW.material_id=311211 BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := repair(db, granter, true, 1000); err == nil {
		t.Fatal("expected a write failure")
	}
	repo := sqlite.New(db, nil)
	for _, user := range before {
		after, err := repo.LoadUser(user.UserId)
		if err != nil || !reflect.DeepEqual(user, after) {
			t.Fatalf("failed repair changed user %d: %v", user.UserId, err)
		}
	}
}

func TestOpenDatabaseNeverCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	for _, apply := range []bool{false, true} {
		db, err := openDatabase(path, apply)
		if err == nil {
			db.Close()
			t.Fatalf("missing database opened with apply=%v", apply)
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("repair created a missing database")
	}
}
