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

func TestMigrateDatabaseAddsBattleBinaryToOlderDatabase(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	statements := []string{
		`ALTER TABLE user_battle DROP COLUMN battle_binary`,
		`DELETE FROM goose_db_version WHERE version_id = 20260805203000`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	if err := migrateDatabase(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query(`PRAGMA table_info(user_battle)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		if name == "battle_binary" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("battle_binary migration was not applied")
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
