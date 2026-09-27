package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func repairFixture(t *testing.T) (string, *sql.DB, *sqlite.SQLiteStore) {
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
	for i := 1; i <= 2; i++ {
		id, err := repo.CreateUser(fmt.Sprintf("parts-%d", i), model.ClientPlatform{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = repo.UpdateUser(id, func(user *store.UserState) {
			for _, key := range []string{"empty-1", "empty-2", "valid", "unknown"} {
				user.Parts[key] = store.PartsState{
					UserPartsUuid: key, PartsId: 36, Level: 1, PartsStatusMainId: 28,
					IsProtected: true, AcquisitionDatetime: 123, LatestVersion: 456,
				}
			}
			part := user.Parts["empty-2"]
			part.Level = 2
			user.Parts["empty-2"] = part
			part = user.Parts["unknown"]
			part.PartsId = 99999999
			user.Parts["unknown"] = part
			user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: "valid", StatusIndex: 1}] = store.PartsStatusSubState{
				UserPartsUuid: "valid", StatusIndex: 1, PartsStatusSubLotteryId: 4, Level: 1,
				StatusKindType: 2, StatusCalculationType: 1, StatusChangeValue: 37, LatestVersion: 456,
			}
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return path, db, repo
}

func runRepair(t *testing.T, path string, extra ...string) (string, error) {
	t.Helper()
	args := []string{"--db", path, "--master", filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")}
	var out bytes.Buffer
	err := run(append(args, extra...), &out)
	return out.String(), err
}

func TestRepairPartsPreviewApplyAndRepeat(t *testing.T) {
	path, _, repo := repairFixture(t)
	before, err := repo.LoadUser(1)
	if err != nil {
		t.Fatal(err)
	}
	output, err := runRepair(t, path, "--user-id", "1")
	if err != nil || !strings.Contains(output, "apply=false candidates=3 repaired=2 skipped=1") || !strings.Contains(output, "total-stats 1 -> 5") {
		t.Fatalf("preview: %s (%v)", output, err)
	}
	after, err := repo.LoadUser(1)
	if err != nil || !maps.Equal(before.Parts, after.Parts) || !maps.Equal(before.PartsStatusSubs, after.PartsStatusSubs) {
		t.Fatalf("preview changed data: %v", err)
	}
	output, err = runRepair(t, path, "--user-id", "1", "--apply")
	if err != nil || !strings.Contains(output, "apply=true candidates=3 repaired=2 skipped=1") {
		t.Fatalf("apply: %s (%v)", output, err)
	}
	after, err = repo.LoadUser(1)
	if err != nil || !maps.Equal(before.Parts, after.Parts) || len(after.PartsStatusSubs) != 9 {
		t.Fatalf("repair changed item identity/main status or did not persist both repairs: %v", err)
	}
	for key, sub := range before.PartsStatusSubs {
		if after.PartsStatusSubs[key] != sub {
			t.Fatal("repair rerolled an existing sub-status")
		}
	}
	catalog, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"empty-1", "empty-2"} {
		used := map[int32]bool{}
		for slot := int32(1); slot <= model.PartsMaxSubStatusCount; slot++ {
			sub := after.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: key, StatusIndex: slot}]
			def, ok := catalog.PartsStatusSubById[sub.PartsStatusSubLotteryId]
			if !ok || used[sub.PartsStatusSubLotteryId] || sub.Level != 1 || sub.LatestVersion <= 456 || sub.StatusKindType != def.StatusKindType || sub.StatusCalculationType != def.StatusCalculationType ||
				sub.StatusChangeValue < def.Initial.Min || sub.StatusChangeValue > def.Initial.Max || (sub.StatusChangeValue-def.Initial.Min)%def.Initial.Step != 0 {
				t.Fatalf("invalid repair: %+v", sub)
			}
			used[sub.PartsStatusSubLotteryId] = true
		}
	}
	other, err := repo.LoadUser(2)
	if err != nil || len(other.PartsStatusSubs) != 1 {
		t.Fatalf("user-id filter changed another player: %v", err)
	}
	output, err = runRepair(t, path, "--user-id", "1", "--apply")
	if err != nil || !strings.Contains(output, "repaired=0 skipped=1") {
		t.Fatalf("repeat: %s (%v)", output, err)
	}
	repeated, err := repo.LoadUser(1)
	if err != nil || !maps.Equal(after.PartsStatusSubs, repeated.PartsStatusSubs) {
		t.Fatalf("repeated repair changed stats: %v", err)
	}
	output, err = runRepair(t, path, "--apply")
	if err != nil || !strings.Contains(output, "repaired=2 skipped=2") {
		t.Fatalf("all players: %s (%v)", output, err)
	}
}

func TestRepairPartsRollsBackOnFailure(t *testing.T) {
	path, db, repo := repairFixture(t)
	_, err := db.Exec(`CREATE TRIGGER fail_repair BEFORE INSERT ON user_parts_status_subs
		WHEN NEW.user_parts_uuid = 'empty-2' AND NEW.status_index = 3 BEGIN SELECT RAISE(ABORT, 'test failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runRepair(t, path, "--apply"); err == nil {
		t.Fatal("repair ignored a failed insert")
	}
	for _, id := range []int64{1, 2} {
		user, err := repo.LoadUser(id)
		if err != nil || len(user.PartsStatusSubs) != 1 {
			t.Fatalf("failed repair partially persisted user %d: %v", id, err)
		}
	}
}

func TestRepairPartsDoesNotCreateMissingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	if _, err := runRepair(t, path, "--apply"); err == nil {
		t.Fatal("repair accepted a nonexistent database")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("repair created missing database: %v", err)
	}
}
