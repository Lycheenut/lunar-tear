package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestConvertExistingMaterialFragments(t *testing.T) {
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
	if err := goose.UpToContext(ctx, db, ".", 20260909130000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (user_id, uuid) VALUES (1, 'fragments'), (2, 'below-threshold');
		INSERT INTO user_materials (user_id, material_id, count) VALUES
			(1, 501001, 25), (1, 322002, 4), (1, 501002, 30), (1, 501003, 20),
			(2, 501001, 9), (2, 501002, 1);
	`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := Up(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		userId     int64
		materialId int32
		count      int32
	}{
		{1, 501001, 5}, {1, 322002, 6}, {1, 312003, 3}, {1, 501003, 20},
		{2, 501001, 9}, {2, 501002, 1},
	} {
		var count int32
		if err := db.QueryRow(`SELECT count FROM user_materials WHERE user_id = ? AND material_id = ?`,
			test.userId, test.materialId).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != test.count {
			t.Fatalf("user %d material %d = %d, want %d", test.userId, test.materialId, count, test.count)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_materials`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 6 {
		t.Fatalf("material rows = %d, want 6 (no empty fragments or unearned materials)", count)
	}
}
