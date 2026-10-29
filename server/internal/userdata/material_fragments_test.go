package userdata

import (
	"context"
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func assertMaterialConversionNotice(t *testing.T, diff *pb.DiffData, userId int64, want map[int32]int32) {
	t.Helper()
	if diff == nil {
		t.Fatal("conversion did not produce the IUserPossessionAutoConvert diff required by the client dialog")
	}
	var records []struct {
		UserId           int64 `json:"userId"`
		PossessionType   int32 `json:"possessionType"`
		PossessionId     int32 `json:"possessionId"`
		FromCount        int32 `json:"fromCount"`
		ToPossessionType int32 `json:"toPossessionType"`
		ToPossessionId   int32 `json:"toPossessionId"`
		ToCount          int32 `json:"toCount"`
		ConvertDatetime  int64 `json:"convertDatetime"`
		LatestVersion    int64 `json:"latestVersion"`
	}
	if err := json.Unmarshal([]byte(diff.UpdateRecordsJson), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != len(want) || diff.DeleteKeysJson != "[]" {
		t.Fatalf("unexpected conversion diff: %+v", diff)
	}
	for _, row := range records {
		target := map[int32]int32{501001: 322002, 501002: 312003}[row.PossessionId]
		count, ok := want[row.PossessionId]
		if !ok || row.UserId != userId || row.PossessionType != int32(model.PossessionTypeMaterial) ||
			row.ToPossessionType != int32(model.PossessionTypeMaterial) || row.ToPossessionId != target ||
			row.FromCount != count*10 || row.ToCount != count || row.ConvertDatetime <= 0 || row.LatestVersion <= 0 {
			t.Fatalf("unexpected conversion record: %+v, want counts %v", row, want)
		}
	}
}

func TestMaterialFragmentConversionPersistsAndSyncs(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("fragments", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	before, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.Materials[501001] = 9
		user.Materials[501002] = 9
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.UpdateUser(userId, func(user *store.UserState) {
		granter := &store.PossessionGranter{}
		granter.GrantFull(user, model.PossessionTypeMaterial, 501001, 1, 1000)
		granter.GrantFull(user, model.PossessionTypeMaterial, 501002, 12, 1000)
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int32]int32{322002: 1, 312003: 2, 501002: 1}
	if !maps.Equal(after.Materials, want) {
		t.Fatalf("persisted materials = %v, want %v", after.Materials, want)
	}
	diffs := ComputeDelta(&before, &after, ChangedTables(&before, &after))
	assertMaterialConversionNotice(t, diffs["IUserPossessionAutoConvert"], userId, map[int32]int32{501001: 1, 501002: 2})
	diff := diffs["IUserMaterial"]
	if diff == nil {
		t.Fatal("conversion did not produce a client material diff")
	}
	var updates []struct {
		MaterialId int32 `json:"materialId"`
		Count      int32 `json:"count"`
	}
	if err := json.Unmarshal([]byte(diff.UpdateRecordsJson), &updates); err != nil {
		t.Fatal(err)
	}
	got := make(map[int32]int32)
	for _, row := range updates {
		got[row.MaterialId] = row.Count
	}
	if !maps.Equal(got, want) {
		t.Fatalf("client updates = %v, want %v", got, want)
	}
	var deletes []struct {
		UserId     int64 `json:"userId"`
		MaterialId int32 `json:"materialId"`
	}
	if err := json.Unmarshal([]byte(diff.DeleteKeysJson), &deletes); err != nil {
		t.Fatal(err)
	}
	if len(deletes) != 1 || deletes[0].UserId != userId || deletes[0].MaterialId != 501001 {
		t.Fatalf("client fragment deletes = %s", diff.DeleteKeysJson)
	}
	if !slices.Contains(AllTableNames(), "IUserPossessionAutoConvert") || projectTable("IUserPossessionAutoConvert", after) != "[]" {
		t.Fatal("full sync must expose the conversion table without replaying old notifications")
	}

	// Repeated and batched grants must show only this request's conversions,
	// including conversions whose products are consumed in the same request.
	for _, grants := range [][]int32{{10}, {10}, {10, 20}, {1}, {0}, {-1}} {
		before = after
		_, err = repo.UpdateUser(userId, func(user *store.UserState) {
			for _, count := range grants {
				store.GrantPossession(user, model.PossessionTypeMaterial, 501001, count)
			}
			store.GrantPossession(user, model.PossessionTypeMaterial, 312003, 1)
			delete(user.Materials, 322002)
		})
		if err != nil {
			t.Fatal(err)
		}
		after, err = repo.LoadUser(userId)
		if err != nil {
			t.Fatal(err)
		}
		notice := ComputeDelta(&before, &after, ChangedTables(&before, &after))["IUserPossessionAutoConvert"]
		if grants[0] < 10 {
			if notice != nil {
				t.Fatalf("non-converting grant replayed notification: %v", notice)
			}
			continue
		}
		var converted int32
		for _, count := range grants {
			converted += count / 10
		}
		assertMaterialConversionNotice(t, notice, userId, map[int32]int32{501001: converted})
	}

	// Snapshot replacement must preserve the counters, or remove them when the
	// imported snapshot does not have them, just like the inventory itself.
	if err := repo.ImportUser(&after); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(reloaded.MaterialConversions, after.MaterialConversions) {
		t.Fatal("snapshot import lost conversion counters")
	}
	after.MaterialConversions = nil
	if err := repo.ImportUser(&after); err != nil {
		t.Fatal(err)
	}
	reloaded, err = repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.MaterialConversions) != 0 {
		t.Fatal("snapshot replacement retained obsolete conversion counters")
	}
}
