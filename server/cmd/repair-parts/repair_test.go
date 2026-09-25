package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func testCatalog() *masterdata.PartsCatalog {
	catalog := &masterdata.PartsCatalog{
		PartsById: map[int32]masterdata.EntityMParts{
			10: {PartsId: 10, RarityType: 10}, 20: {PartsId: 20, RarityType: 40},
		},
		RarityByRarityType: map[model.RarityType]masterdata.EntityMPartsRarity{
			10: {PartsLevelUpPriceGroupId: 1}, 40: {PartsLevelUpPriceGroupId: 2},
		},
		PriceByGroupAndLevel: map[int32]map[int32]int32{1: {}, 2: {}},
		PartsStatusSubById: map[int32]model.PartsStatusSubDef{
			4:  {StatusKindType: 2, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 30, Max: 45, Step: 1}},
			12: {StatusKindType: 2, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 15, Max: 25, Step: 5}},
			24: {StatusKindType: 6, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 650, Max: 750, Step: 5}},
			28: {StatusKindType: 4, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 30, Max: 50, Step: 5}},
		},
	}
	for level := int32(1); level < model.PartsMaxLevel; level++ {
		catalog.PriceByGroupAndLevel[1][level] = 10 * level
		catalog.PriceByGroupAndLevel[2][level] = 100 * level
	}
	return catalog
}

func fixture(t *testing.T) (*sql.DB, string, []store.UserState) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "game #1.db")
	db, err := database.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	catalog := testCatalog()
	for id := int64(1); id <= 3; id++ {
		user := store.SeedUserState(id, fmt.Sprintf("repair-%d", id), 10, model.ClientPlatform{})
		user.PlayerId = 100 + id
		user.ConsumableItems[88] = 123 // Unrelated currency.
		delete(user.ConsumableItems, 99)
		if id == 1 {
			user.ConsumableItems[99] = 1000
		}
		user.Parts["untouched"] = store.PartsState{UserPartsUuid: "untouched", PartsId: 20, Level: 1, LatestVersion: 10}
		key := store.PartsStatusSubKey{UserPartsUuid: "untouched", StatusIndex: 1}
		user.PartsStatusSubs[key] = store.PartsStatusSubState{UserPartsUuid: "untouched", StatusIndex: 1, Level: 1, StatusChangeValue: 999, LatestVersion: 10}
		if id <= 2 {
			// UUIDs are only unique within a player. User 2 has no sub-statuses.
			user.Parts["shared"] = store.PartsState{
				UserPartsUuid: "shared", PartsId: 10, Level: int32(id * 2), PartsStatusMainId: 9,
				IsProtected: true, AcquisitionDatetime: 5, LatestVersion: 10,
			}
			user.DeckParts["character"] = []string{"shared"}
			user.PartsPresets[1] = store.PartsPresetState{UserPartsPresetNumber: 1, UserPartsUuid01: "shared", Name: "keep", LatestVersion: 10}
		}
		if id == 1 {
			user.Parts["max"] = store.PartsState{UserPartsUuid: "max", PartsId: 20, Level: 15, PartsStatusMainId: 28, LatestVersion: 10}
			for i, lotteryID := range []int32{4, 12, 24, 28} {
				def := catalog.PartsStatusSubById[lotteryID]
				for _, uuid := range []string{"shared", "max"} {
					slot := int32(i + 1)
					key := store.PartsStatusSubKey{UserPartsUuid: uuid, StatusIndex: slot}
					user.PartsStatusSubs[key] = store.PartsStatusSubState{
						UserPartsUuid: uuid, StatusIndex: slot, PartsStatusSubLotteryId: lotteryID,
						Level: 1 + int32(i)*3, StatusKindType: def.StatusKindType, StatusCalculationType: def.StatusCalculationType,
						StatusChangeValue: 9999, LatestVersion: 10,
					}
				}
			}
		}
		if err := repo.ImportUser(user); err != nil {
			t.Fatal(err)
		}
	}
	return db, path, loadUsers(t, db)
}

func loadUsers(t *testing.T, db *sql.DB) []store.UserState {
	t.Helper()
	users := []store.UserState{}
	for id := int64(1); id <= 3; id++ {
		user, err := sqlite.New(db, nil).LoadUser(id)
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, user)
	}
	return users
}

func assertUsers(t *testing.T, db *sql.DB, want []store.UserState) {
	t.Helper()
	for i, user := range loadUsers(t, db) {
		if !reflect.DeepEqual(user, want[i]) {
			t.Fatalf("unexpected changes to player %d", user.PlayerId)
		}
	}
}

func TestRepairPreviewApplyAndRepeat(t *testing.T) {
	db, path, before := fixture(t)
	catalog := testCatalog()
	for _, apply := range []bool{false, true} {
		connection, err := openDatabase(path, apply)
		if err != nil {
			t.Fatal(err)
		}
		report, err := repair(connection, catalog, 99, apply, 1000)
		connection.Close()
		if err != nil {
			t.Fatal(err)
		}
		if report.Applied != apply || report.ScannedPlayers != 3 || report.ResetParts != 3 ||
			report.GoldItemID != 99 || report.GoldRefund != 10570 || len(report.Players) != 2 {
			t.Fatalf("unexpected report: %+v", report)
		}
		if report.Players[0].GoldBefore != 1000 || report.Players[0].GoldRefund != 10510 || report.Players[0].GoldAfter != 11510 ||
			report.Players[1].GoldBefore != 0 || report.Players[1].GoldRefund != 60 || report.Players[1].GoldAfter != 60 {
			t.Fatalf("incorrect refunds: %+v", report.Players)
		}
		want := append([]store.UserState(nil), before...)
		for _, player := range report.Players {
			user := &want[player.UserID-1]
			user.Parts = maps.Clone(user.Parts)
			user.PartsStatusSubs = maps.Clone(user.PartsStatusSubs)
			user.ConsumableItems = maps.Clone(user.ConsumableItems)
			user.ConsumableItems[99] = int32(player.GoldAfter)
			for _, part := range player.Parts {
				wantSubCount := 0
				if player.UserID == 1 {
					wantSubCount = 4
				}
				if len(part.SubStatuses) != wantSubCount {
					t.Fatalf("part %s has %d sub-status changes, want %d", part.UUID, len(part.SubStatuses), wantSubCount)
				}
				old := user.Parts[part.UUID]
				if part.LevelBefore != old.Level || part.LevelAfter != 1 {
					t.Fatalf("unexpected level change: %+v", part)
				}
				old.Level, old.LatestVersion = 1, 1000
				user.Parts[part.UUID] = old
				for _, sub := range part.SubStatuses {
					key := store.PartsStatusSubKey{UserPartsUuid: part.UUID, StatusIndex: sub.StatusIndex}
					old := user.PartsStatusSubs[key]
					rangeDef := catalog.PartsStatusSubById[old.PartsStatusSubLotteryId].Initial
					if sub.LevelBefore != old.Level || sub.LevelAfter != 1 || sub.ValueBefore != old.StatusChangeValue ||
						sub.PartsStatusSubLotteryID != old.PartsStatusSubLotteryId || sub.StatusKindType != old.StatusKindType ||
						sub.StatusCalculationType != old.StatusCalculationType || sub.ValueAfter < rangeDef.Min ||
						sub.ValueAfter > rangeDef.Max || (sub.ValueAfter-rangeDef.Min)%rangeDef.Step != 0 {
						t.Fatalf("incorrect sub-status reroll: %+v", sub)
					}
					old.Level, old.StatusChangeValue, old.LatestVersion = 1, sub.ValueAfter, 1000
					user.PartsStatusSubs[key] = old
				}
			}
		}
		if !apply {
			assertUsers(t, db, before)
			continue
		}
		assertUsers(t, db, want)
		repeated, err := repair(db, catalog, 99, true, 2000)
		if err != nil || !repeated.Applied || repeated.ResetParts != 0 || repeated.GoldRefund != 0 || len(repeated.Players) != 0 {
			t.Fatalf("repeat = %+v, error = %v", repeated, err)
		}
		assertUsers(t, db, want)
	}
}

func TestRepairErrorsLeaveAllPlayersUnchanged(t *testing.T) {
	for _, tc := range []struct {
		name, sql, want string
		changeCatalog   func(*masterdata.PartsCatalog)
	}{
		{name: "unknown part", sql: `UPDATE user_parts SET parts_id = 999 WHERE user_id = 2 AND level > 1`, want: "unknown parts ID"},
		{name: "invalid level", sql: `UPDATE user_parts SET level = 16 WHERE user_id = 2 AND level > 1`, want: "invalid enhanced level"},
		{name: "unknown sub", sql: `UPDATE user_parts_status_subs SET parts_status_sub_lottery_id = 999 WHERE user_parts_uuid = 'max'`, want: "unknown or mismatched"},
		{name: "mismatched type", sql: `UPDATE user_parts_status_subs SET status_calculation_type = 99 WHERE user_parts_uuid = 'max'`, want: "unknown or mismatched"},
		{name: "missing rarity", want: "unknown rarity", changeCatalog: func(c *masterdata.PartsCatalog) { delete(c.RarityByRarityType, 40) }},
		{name: "missing price", want: "enhancement price", changeCatalog: func(c *masterdata.PartsCatalog) { delete(c.PriceByGroupAndLevel[1], 3) }},
		{name: "negative price", want: "enhancement price", changeCatalog: func(c *masterdata.PartsCatalog) { c.PriceByGroupAndLevel[1][3] = -1 }},
		{name: "overflow", sql: fmt.Sprintf(`INSERT INTO user_consumable_items VALUES (2, 99, %d)`, math.MaxInt32-59), want: "overflow"},
		{name: "write failure", sql: `CREATE TRIGGER fail_repair BEFORE INSERT ON user_consumable_items
			WHEN NEW.user_id = 2 AND NEW.consumable_item_id = 99 BEGIN SELECT RAISE(ABORT, 'injected failure'); END`, want: "injected failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, path, _ := fixture(t)
			if tc.sql != "" {
				if _, err := db.Exec(tc.sql); err != nil {
					t.Fatal(err)
				}
			}
			before := loadUsers(t, db)
			catalog := testCatalog()
			if tc.changeCatalog != nil {
				tc.changeCatalog(catalog)
			}
			connection, err := openDatabase(path, true)
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			report, err := repair(connection, catalog, 99, true, 1000)
			if err == nil || !strings.Contains(err.Error(), tc.want) || report.Applied {
				t.Fatalf("report = %+v, error = %v; want %q", report, err, tc.want)
			}
			assertUsers(t, db, before)
		})
	}
}

func TestMissingDatabaseIsNeverCreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	for _, apply := range []bool{false, true} {
		if db, err := openDatabase(path, apply); err == nil {
			db.Close()
			t.Fatal("missing database was opened")
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("missing database was created: %v", err)
	}
}

func TestRunPreviewWithMasterData(t *testing.T) {
	masterPath := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(masterPath); os.IsNotExist(err) {
		t.Skip("local master data not installed")
	}
	if err := memorydb.Init(masterPath); err != nil {
		t.Fatal(err)
	}
	catalog, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	var partsID int32
	for id, part := range catalog.PartsById {
		if part.RarityType == 40 && (partsID == 0 || id < partsID) {
			partsID = id
		}
	}
	if partsID == 0 {
		t.Fatal("master data has no four-star memoirs")
	}
	db, path, _ := fixture(t)
	if _, err := db.Exec(`UPDATE user_parts SET level = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE user_parts SET parts_id = ?, level = 15 WHERE user_id = 1 AND user_parts_uuid = 'max'`, partsID); err != nil {
		t.Fatal(err)
	}
	before := loadUsers(t, db)
	var stdout bytes.Buffer
	if err := run([]string{"--db", path, "--master-data", masterPath}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	var report repairReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Applied || report.ScannedPlayers != 3 || report.ResetParts != 1 || report.GoldItemID <= 0 ||
		report.GoldRefund <= 0 || len(report.Players) != 1 || len(report.Players[0].Parts[0].SubStatuses) != 4 {
		t.Fatalf("unexpected CLI report: %+v", report)
	}
	for _, sub := range report.Players[0].Parts[0].SubStatuses {
		r := catalog.PartsStatusSubById[sub.PartsStatusSubLotteryID].Initial
		if sub.ValueAfter < r.Min || sub.ValueAfter > r.Max || (sub.ValueAfter-r.Min)%r.Step != 0 {
			t.Fatalf("CLI reroll outside current initial range: %+v", sub)
		}
	}
	assertUsers(t, db, before)
}

func TestRepairRequiresGoldConfig(t *testing.T) {
	db, _, before := fixture(t)
	if _, err := repair(db, testCatalog(), 0, true, 1000); err == nil {
		t.Fatal("missing gold config accepted")
	}
	assertUsers(t, db, before)
}
