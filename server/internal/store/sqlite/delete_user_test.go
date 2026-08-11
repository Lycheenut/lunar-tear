package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/migrations"
)

func TestDeleteUserRemovesOwnedAndInboundSocialState(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := New(db, nil)
	deletedID, err := repo.CreateUser("deleted", model.DefaultPlatform)
	if err != nil {
		t.Fatal(err)
	}
	remainingID, err := repo.CreateUser("remaining", model.DefaultPlatform)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateUsers([]int64{deletedID, remainingID}, func(users map[int64]*store.UserState) error {
		users[deletedID].Friends[remainingID] = store.FriendState{IsFriend: true}
		users[remainingID].Friends[deletedID] = store.FriendState{IsFriend: true}
		users[remainingID].FriendRequests[deletedID] = 1
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteUser(deletedID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.LoadUser(deletedID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("LoadUser() after delete error = %v, want ErrNotFound", err)
	}
	remaining, err := repo.LoadUser(remainingID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := remaining.Friends[deletedID]; ok {
		t.Fatal("inbound friendship survived deletion")
	}
	if _, ok := remaining.FriendRequests[deletedID]; ok {
		t.Fatal("outgoing friend request survived deletion")
	}

	for _, table := range UserOwnedTables() {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM `+table+` WHERE user_id = ?`, deletedID).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("%s still has %d rows for deleted user", table, count)
		}
	}
	if err := repo.DeleteUser(deletedID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second DeleteUser() error = %v, want ErrNotFound", err)
	}
}
