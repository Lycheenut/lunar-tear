package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestAlignDailyGachaIdPreservesLatestDailyDraws(t *testing.T) {
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
	if err := goose.UpToContext(ctx, db, ".", 20260909120000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (user_id, uuid) VALUES
			(1, 'legacy'), (2, 'legacy-newer'), (3, 'canonical-newer'), (4, 'same-day'), (5, 'canonical-only');
		INSERT INTO user_gacha_banners (user_id, gacha_id, draw_count) VALUES
			(1, 50001, 5), (1, 45, 10),
			(2, 50001, 5), (2, 50030, 0),
			(3, 50001, 5), (3, 50030, 0),
			(4, 50001, 5), (4, 50030, 0),
			(5, 50030, 5);
		INSERT INTO user_gacha_banner_box_drew_counts (user_id, gacha_id, box_item_id, count) VALUES
			(1, 50001, -2, 20260909), (1, 45, 1, 10),
			(2, 50001, -2, 20260909), (2, 50030, -2, 20260908),
			(3, 50001, -2, 20260908), (3, 50030, -2, 20260909),
			(4, 50001, -2, 20260909), (4, 50030, -2, 20260909),
			(5, 50030, -2, 20260909);
	`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := Up(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct{ userId, drawCount int }{{1, 5}, {2, 5}, {3, 0}, {4, 5}, {5, 5}} {
		var draws, day int
		if err := db.QueryRow(`
			SELECT b.draw_count, d.count FROM user_gacha_banners b
			JOIN user_gacha_banner_box_drew_counts d USING (user_id, gacha_id)
			WHERE b.user_id = ? AND b.gacha_id = 50030 AND d.box_item_id = -2
		`, tt.userId).Scan(&draws, &day); err != nil {
			t.Fatal(err)
		}
		if draws != tt.drawCount || day != 20260909 {
			t.Fatalf("user %d: draws/day = %d/%d, want %d/20260909", tt.userId, draws, day, tt.drawCount)
		}
	}
	for _, query := range []string{
		`SELECT COUNT(*) FROM user_gacha_banners WHERE gacha_id = 50001`,
		`SELECT COUNT(*) FROM user_gacha_banner_box_drew_counts WHERE gacha_id = 50001`,
	} {
		var count int
		if err := db.QueryRow(query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("legacy state remains: count=%d, err=%v", count, err)
		}
	}
	var draws, counter int
	if err := db.QueryRow(`
		SELECT b.draw_count, d.count FROM user_gacha_banners b
		JOIN user_gacha_banner_box_drew_counts d USING (user_id, gacha_id)
		WHERE b.user_id = 1 AND b.gacha_id = 45 AND d.box_item_id = 1
	`).Scan(&draws, &counter); err != nil || draws != 10 || counter != 10 {
		t.Fatalf("unrelated Gacha changed: draws/counter=%d/%d, err=%v", draws, counter, err)
	}
}
