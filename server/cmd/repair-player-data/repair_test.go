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
		if after.Characters[1001].Level != 1 {
			t.Fatal("reward costume did not initialize character 1001")
		}
		for uuid, costume := range after.Costumes {
			if costume.CostumeId == 24008 && after.CostumeActiveSkills[uuid].Level != 1 {
				t.Fatal("reward costume is missing its active skill")
			}
		}
		for uuid, weapon := range after.Weapons {
			if weapon.WeaponId != 240271 {
				continue
			}
			wantSkills := []store.WeaponSkillState{
				{UserWeaponUuid: uuid, SlotNumber: 1, Level: 1},
				{UserWeaponUuid: uuid, SlotNumber: 2, Level: 1},
			}
			wantAbilities := []store.WeaponAbilityState{
				{UserWeaponUuid: uuid, SlotNumber: 1, Level: 1},
				{UserWeaponUuid: uuid, SlotNumber: 3, Level: 1},
			}
			if !reflect.DeepEqual(after.WeaponSkills[uuid], wantSkills) || !reflect.DeepEqual(after.WeaponAbilities[uuid], wantAbilities) {
				t.Fatal("reward weapon skills or abilities differ from the server grant")
			}
		}
		if after.WeaponStories[240271].ReleasedMaxStoryIndex != 1 || after.WeaponNotes[240271].MaxLevel != 1 {
			t.Fatal("reward weapon story or collection record was not initialized")
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

func TestRepairPreservesDuplicateCostumeCompensation(t *testing.T) {
	granter, err := loadGranter(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	db, _, users := fixture(t)
	repo := sqlite.New(db, nil)
	before, err := repo.UpdateUser(users[1].UserId, func(user *store.UserState) {
		granter.GrantCostume(user, 24008, 100)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repair(db, granter, true, 1000); err != nil {
		t.Fatal(err)
	}
	after, err := repo.LoadUser(before.UserId)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Costumes) != len(before.Costumes) {
		t.Fatal("repair added a duplicate costume")
	}
	// The existing server grant gives 10 books and 1 awakening stone for the
	// duplicate costume, in addition to the bundle's 40 books and 5 stones.
	for id, count := range map[int32]int32{311211: 50, 313197: 6, 312011: 4} {
		if got := after.Materials[id] - before.Materials[id]; got != count {
			t.Errorf("material %d: got delta %d, want %d", id, got, count)
		}
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

func TestRepairCompanionsUsesLoginUnlockAndPreservesOwned(t *testing.T) {
	db, path, users := fixture(t)
	granter, err := loadGranter(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	for i, companions := range map[int]map[int32]int32{
		0: {2: 1}, // Owning a companion cannot bypass the login unlock gate.
		2: {1: 10, 31: 37, 49: 1, 50: 25, 51: 50, 53: 15},
	} {
		if _, err := repo.UpdateUser(users[i].UserId, func(user *store.UserState) {
			for id, level := range companions {
				granter.GrantCompanion(user, id, 100)
				for key, companion := range user.Companions {
					if companion.CompanionId == id {
						companion.Level = level
						user.Companions[key] = companion
					}
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i, user := range users {
		users[i], err = repo.LoadUser(user.UserId)
		if err != nil {
			t.Fatal(err)
		}
	}
	readOnly, err := openDatabase(path, false)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := repair(readOnly, granter, false, 1000)
	readOnly.Close()
	if err != nil || len(preview.Players) != 2 {
		t.Fatalf("preview=%+v error=%v", preview, err)
	}
	for _, user := range users {
		after, err := repo.LoadUser(user.UserId)
		if err != nil || !reflect.DeepEqual(user, after) {
			t.Fatalf("preview changed player %d: %v", user.UserId, err)
		}
	}
	applied, err := repair(db, granter, true, 1000)
	if err != nil || !reflect.DeepEqual(preview.Players, applied.Players) {
		t.Fatalf("apply differs from preview: report=%+v error=%v", applied, err)
	}
	for i, user := range users {
		after, err := repo.LoadUser(user.UserId)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if !reflect.DeepEqual(user, after) {
				t.Fatal("locked player with a companion was modified")
			}
			continue
		}
		wantCount := 23
		if i == 2 {
			wantCount++ // Preserve the existing main-quest companion too.
		}
		if len(after.Companions) != wantCount || !reflect.DeepEqual(user.ConsumableItems, after.ConsumableItems) {
			t.Fatal("expected unique companions without duplicate compensation")
		}
		if !reflect.DeepEqual(user.Tutorials, after.Tutorials) {
			t.Fatal("companion reward changed tutorial progress")
		}
		for _, companion := range after.Companions {
			if companion.CompanionId >= 49 && companion.CompanionId <= 51 && companion.Level != 50 {
				t.Fatalf("companion not granted or repaired at level 50: %+v", companion)
			}
		}
		for key, original := range user.Companions {
			got, exists := after.Companions[key]
			wantLevel := original.Level
			if original.CompanionId >= 49 && original.CompanionId <= 51 {
				wantLevel = 50
			}
			if !exists || got.Level != wantLevel || got.AcquisitionDatetime != original.AcquisitionDatetime {
				t.Fatalf("existing companion not preserved: original=%+v got=%+v", original, got)
			}
		}
	}
	for _, player := range preview.Players {
		if player.UserID == users[2].UserId {
			foundRepair := false
			for _, level := range player.CompanionLevels {
				if level.ID == 49 && level.Before == 1 && level.After == 50 {
					foundRepair = true
				}
			}
			if !foundRepair {
				t.Fatal("preview omitted level-only repair")
			}
		}
	}
	repeat, err := repair(db, granter, false, 2000)
	if err != nil {
		t.Fatal(err)
	}
	for _, player := range repeat.Players {
		if len(player.CompanionLevels) != 0 {
			t.Fatal("repeated repair would change companion levels")
		}
		for _, change := range player.Inventory {
			if change.Type == int32(model.PossessionTypeCompanion) {
				t.Fatal("repeated repair would grant another companion")
			}
		}
	}
}

func TestRepairCompanionsRollsBackAllPlayersOnWriteFailure(t *testing.T) {
	db, _, users := fixture(t)
	granter := &store.PossessionGranter{
		CostumeById: map[int32]store.CostumeRef{24008: {CharacterId: 24}},
		WeaponById:  map[int32]store.WeaponRef{240271: {}},
	}
	repo := sqlite.New(db, nil)
	for i, user := range users {
		var err error
		users[i], err = repo.UpdateUser(user.UserId, func(user *store.UserState) { granter.GrantCompanion(user, 2, 100) })
		if err != nil {
			t.Fatal(err)
		}
		users[i], err = repo.LoadUser(user.UserId)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`CREATE TRIGGER reject_companion_backfill BEFORE INSERT ON user_companions
		WHEN NEW.user_id=3 AND NEW.companion_id=49 BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := repair(db, granter, true, 1000); err == nil {
		t.Fatal("expected companion write failure")
	}
	for _, user := range users {
		after, err := repo.LoadUser(user.UserId)
		if err != nil || !reflect.DeepEqual(user, after) {
			t.Fatalf("failed companion repair changed player %d: %v", user.UserId, err)
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
