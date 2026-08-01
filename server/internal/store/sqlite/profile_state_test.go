package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/migrations"
)

func TestProfileStateRoundTrip(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := New(db, nil)
	userId, err := repo.CreateUser("profile", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.Profile.CurrentPvpRank = 11
		user.Profile.CurrentPvpGradeId = 12
		user.Profile.MaxPvpSeasonRank = 13
	}); err != nil {
		t.Fatal(err)
	}

	user, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if user.Profile.CurrentPvpRank != 11 || user.Profile.CurrentPvpGradeId != 12 || user.Profile.MaxPvpSeasonRank != 13 {
		t.Fatalf("pvp profile = %+v", user.Profile)
	}
	foundUserId, err := repo.GetUserByPlayerId(user.PlayerId)
	if err != nil {
		t.Fatal(err)
	}
	if foundUserId != userId {
		t.Fatalf("user id = %d, want %d", foundUserId, userId)
	}
}
