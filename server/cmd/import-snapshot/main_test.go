package main

import (
	"context"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestResolveTargetUUIDUsesExistingLocalUser(t *testing.T) {
	repo := newTestStore(t)
	userID, err := repo.CreateUser("local-client-uuid", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}

	snapshot := store.UserState{UserId: userID, Uuid: "production-uuid"}
	if err := resolveTargetUUID(repo, &snapshot, ""); err != nil {
		t.Fatal(err)
	}
	if snapshot.Uuid != "local-client-uuid" {
		t.Fatalf("got UUID %q", snapshot.Uuid)
	}
}

func TestResolveTargetUUIDAcceptsExplicitValue(t *testing.T) {
	repo := newTestStore(t)
	snapshot := store.UserState{UserId: 99, Uuid: "production-uuid"}
	if err := resolveTargetUUID(repo, &snapshot, "explicit-local-uuid"); err != nil {
		t.Fatal(err)
	}
	if snapshot.Uuid != "explicit-local-uuid" {
		t.Fatalf("got UUID %q", snapshot.Uuid)
	}
}

func TestResolveTargetUUIDRequiresValueForMissingUser(t *testing.T) {
	repo := newTestStore(t)
	snapshot := store.UserState{UserId: 99}
	if err := resolveTargetUUID(repo, &snapshot, ""); err == nil {
		t.Fatal("expected an error for a missing local user")
	}
}

func newTestStore(t *testing.T) *sqlite.SQLiteStore {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return sqlite.New(db, nil)
}
