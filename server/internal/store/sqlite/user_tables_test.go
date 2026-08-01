package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/migrations"
)

func TestUserOwnedTablesCoverSchema(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	listed := make(map[string]bool, len(userOwnedTables))
	for _, table := range userOwnedTables {
		if listed[table] {
			t.Fatalf("duplicate user-owned table %q", table)
		}
		listed[table] = true
	}

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := make(map[string]bool)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatal(err)
		}
		if table == "users" {
			continue
		}
		columns, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%q)`, table))
		if err != nil {
			t.Fatal(err)
		}
		hasUserID := false
		for columns.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue any
			if err := columns.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				columns.Close()
				t.Fatal(err)
			}
			if name == "user_id" {
				hasUserID = true
			}
		}
		columns.Close()
		if hasUserID {
			seen[table] = true
			if !listed[table] {
				t.Errorf("user-owned table %q is missing from UserOwnedTables", table)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for table := range listed {
		if !seen[table] {
			t.Errorf("listed user-owned table %q does not exist or lacks user_id", table)
		}
	}
}
