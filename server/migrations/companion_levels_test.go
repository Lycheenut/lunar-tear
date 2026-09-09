package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestRepairUnenhanceableCompanionLevels(t *testing.T) {
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
	if err := goose.UpToContext(ctx, db, ".", 20260903170000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (user_id, uuid) VALUES (1, 'companion-player'), (2, 'max-level-player');
		INSERT INTO user_companions (
			user_id, user_companion_uuid, companion_id, level,
			headup_display_view_id, acquisition_datetime, latest_version
		) VALUES
			(1, 'mama', 49, 1, 2, 1000, 2000),
			(1, 'carrier', 50, 1, 2, 1000, 2000),
			(1, 'riding-hood', 51, 20, 2, 1000, 2000),
			(2, 'max-level', 49, 50, 2, 1000, 2000),
			(1, 'ordinary', 35, 12, 2, 1000, 2000),
			(1, 'dracky', 53, 1, 2, 1000, 2000);
	`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := Up(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		uuid     string
		level    int
		repaired bool
	}{
		{"mama", 50, true}, {"carrier", 50, true}, {"riding-hood", 50, true},
		{"max-level", 50, false}, {"ordinary", 12, false}, {"dracky", 1, false},
	} {
		var level, display, acquired, version int64
		if err := db.QueryRow(`
			SELECT level, headup_display_view_id, acquisition_datetime, latest_version
			FROM user_companions WHERE user_companion_uuid = ?
		`, test.uuid).Scan(&level, &display, &acquired, &version); err != nil {
			t.Fatal(err)
		}
		if level != int64(test.level) || display != 2 || acquired != 1000 {
			t.Fatalf("%s: level/display/acquired = %d/%d/%d", test.uuid, level, display, acquired)
		}
		if (test.repaired && version <= 2000) || (!test.repaired && version != 2000) {
			t.Fatalf("%s: latest version = %d, repaired = %v", test.uuid, version, test.repaired)
		}
	}
}
