package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func main() {
	dbPath := flag.String("db", "db/game.db", "SQLite database path")
	snapshotPath := flag.String("snapshot", "", "Path to JSON snapshot file (required)")
	userUuid := flag.String("uuid", "", "UUID to assign (default: keep the existing local user's UUID)")
	flag.Parse()

	if *snapshotPath == "" {
		log.Fatal("--snapshot flag is required")
	}
	data, err := os.ReadFile(*snapshotPath)
	if err != nil {
		log.Fatalf("read snapshot: %v", err)
	}
	log.Printf("read %d bytes from %s", len(data), *snapshotPath)

	var u store.UserState
	if err := json.Unmarshal(data, &u); err != nil {
		log.Fatalf("unmarshal snapshot: %v", err)
	}
	u.EnsureMaps()

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if err := migrateDatabase(context.Background(), db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	userStore := sqlite.New(db, nil)
	if err := resolveTargetUUID(userStore, &u, *userUuid); err != nil {
		log.Fatal(err)
	}

	log.Printf("parsed user %d (uuid=%s, costumes=%d, weapons=%d, characters=%d, quests=%d)",
		u.UserId, u.Uuid, len(u.Costumes), len(u.Weapons), len(u.Characters), len(u.Quests))

	if err := userStore.ImportUser(&u); err != nil {
		log.Fatalf("import user: %v", err)
	}

	log.Printf("imported user %d successfully", u.UserId)
}

func migrateDatabase(ctx context.Context, db *sql.DB) error {
	return migrations.Up(ctx, db)
}

func resolveTargetUUID(userStore *sqlite.SQLiteStore, user *store.UserState, provided string) error {
	if provided != "" {
		user.Uuid = provided
		return nil
	}

	local, err := userStore.LoadUser(user.UserId)
	if err != nil {
		return fmt.Errorf("--uuid is required because local user %d does not exist: %w", user.UserId, err)
	}
	user.Uuid = local.Uuid
	return nil
}
