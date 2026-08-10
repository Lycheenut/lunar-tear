package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestCampaignLoginBonusMigrationPreservesExistingProgress(t *testing.T) {
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
	if err := goose.UpToContext(ctx, db, ".", 20260805203000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (user_id, uuid) VALUES (1, 'existing')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_login_bonus (
		user_id, login_bonus_id, current_page_number, current_stamp_number,
		latest_reward_receive_datetime, latest_version
	) VALUES (1, 1, 2, 3, 4, 5)`); err != nil {
		t.Fatal(err)
	}

	if err := Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_login_bonus (
		user_id, login_bonus_id, current_page_number, current_stamp_number,
		latest_reward_receive_datetime, latest_version
	) VALUES (1, 91, 1, 0, 0, 6)`); err != nil {
		t.Fatalf("composite login bonus key was not installed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO user_beginner_campaign (
		user_id, beginner_campaign_id, campaign_register_datetime, latest_version
	) VALUES (1, 1, 10, 11)`); err != nil {
		t.Fatalf("beginner campaign table was not installed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO user_comeback_campaign (
		user_id, comeback_campaign_id, comeback_datetime, latest_version
	) VALUES (1, 2, 20, 21)`); err != nil {
		t.Fatalf("comeback campaign table was not installed: %v", err)
	}

	var count, page, stamp int
	if err := db.QueryRow(`SELECT count(*), max(current_page_number), max(current_stamp_number)
		FROM user_login_bonus WHERE user_id=1`).Scan(&count, &page, &stamp); err != nil {
		t.Fatal(err)
	}
	if count != 2 || page != 2 || stamp != 3 {
		t.Fatalf("migrated login bonuses = count %d page %d stamp %d, want 2/2/3", count, page, stamp)
	}
}
