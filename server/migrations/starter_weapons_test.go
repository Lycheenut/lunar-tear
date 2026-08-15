package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestRestoreStarterWeaponsMigration(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	goose.SetBaseFS(FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := goose.UpToContext(ctx, db, ".", 20260813140000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (user_id, uuid, game_start_datetime) VALUES
			(1, 'missing-all', 1000),
			(2, 'missing-two', 2000);
		INSERT INTO user_weapons (
			user_id, user_weapon_uuid, weapon_id, level, acquisition_datetime
		) VALUES (2, 'existing', 100001, 7, 1500);
		INSERT INTO user_weapon_notes (
			user_id, weapon_id, max_level, max_limit_break_count,
			first_acquisition_datetime, latest_version
		) VALUES (2, 100011, 17, 2, 1600, 1700);
		INSERT INTO user_weapon_stories (
			user_id, weapon_id, released_max_story_index, latest_version
		) VALUES (2, 100011, 3, 1700);
	`); err != nil {
		t.Fatal(err)
	}

	if err := Up(ctx, db); err != nil {
		t.Fatal(err)
	}

	assertCount := func(query string, want int) {
		t.Helper()
		var got int
		if err := db.QueryRow(query).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("count for %q = %d, want %d", query, got, want)
		}
	}
	assertCount(`SELECT count(*) FROM user_weapons WHERE weapon_id IN (100001, 100011, 100021)`, 6)
	assertCount(`SELECT count(*) FROM user_weapon_skills WHERE user_weapon_uuid <> 'existing'`, 10)
	assertCount(`SELECT count(*) FROM user_weapon_abilities WHERE user_weapon_uuid <> 'existing'`, 5)

	var level, acquired, protected int
	if err := db.QueryRow(`
		SELECT level, acquisition_datetime, is_protected
		FROM user_weapons
		WHERE user_id = 1 AND weapon_id = 100001
	`).Scan(&level, &acquired, &protected); err != nil {
		t.Fatal(err)
	}
	if level != 1 || acquired != 1000 || protected != 0 {
		t.Fatalf("restored weapon = level %d acquired %d protected %d, want 1/1000/0", level, acquired, protected)
	}

	var maxLevel, storyIndex int
	if err := db.QueryRow(`SELECT max_level FROM user_weapon_notes WHERE user_id = 2 AND weapon_id = 100011`).Scan(&maxLevel); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT released_max_story_index FROM user_weapon_stories WHERE user_id = 2 AND weapon_id = 100011`).Scan(&storyIndex); err != nil {
		t.Fatal(err)
	}
	if maxLevel != 17 || storyIndex != 3 {
		t.Fatalf("historical progress changed: max level %d, story %d", maxLevel, storyIndex)
	}
}
