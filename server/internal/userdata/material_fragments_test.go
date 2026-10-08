package userdata

import (
	"context"
	"encoding/json"
	"maps"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

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
	diff := ComputeDelta(&before, &after, ChangedTables(&before, &after))["IUserMaterial"]
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
}
