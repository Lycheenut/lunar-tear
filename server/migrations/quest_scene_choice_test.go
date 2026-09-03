package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestQuestSceneChoiceMigrationUsesClientKeysAndRepairsLegacyRows(t *testing.T) {
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
	if err := goose.UpToContext(ctx, db, ".", 20260816120000); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (user_id, uuid) VALUES (2, 'ending-player');
		INSERT INTO user_quest_scene_choices (
			user_id, quest_scene_id, quest_flow_type, choice_number, choice_datetime, latest_version
		) VALUES
			(2, 1113, 1, 1, 100, 100),
			(2, 1113, 3, 2, 200, 200),
			(2, 1713, 4, 3, 300, 300),
			(2, 9999, 3, 3, 999, 999);
		INSERT INTO user_quest_scene_choice_history (
			user_id, quest_scene_id, quest_flow_type, choice_number, choice_datetime, latest_version
		) VALUES
			(2, 1113, 1, 1, 100, 100),
			(2, 1713, 3, 1, 400, 400),
			(2, 1113, 3, 2, 200, 200),
			(2, 9999, 3, 3, 999, 999);
	`); err != nil {
		t.Fatal(err)
	}

	if err := Up(ctx, db); err != nil {
		t.Fatal(err)
	}

	var groupingId, effectId, latestVersion int
	if err := db.QueryRow(`
		SELECT quest_scene_choice_grouping_id, quest_scene_choice_effect_id, latest_version
		FROM user_quest_scene_choices WHERE user_id = 2
	`).Scan(&groupingId, &effectId, &latestVersion); err != nil {
		t.Fatal(err)
	}
	if groupingId != 1 || effectId != 3 || latestVersion != 300 {
		t.Fatalf("current choice = grouping %d effect %d version %d, want 1/3/300", groupingId, effectId, latestVersion)
	}

	rows, err := db.Query(`
		SELECT quest_scene_choice_effect_id, choice_datetime, latest_version
		FROM user_quest_scene_choice_history WHERE user_id = 2
		ORDER BY quest_scene_choice_effect_id
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := [][3]int{{1, 100, 400}, {2, 200, 200}}
	count := 0
	for rows.Next() {
		i := count
		if i >= len(want) {
			t.Fatal("migration kept an invalid history row")
		}
		var got [3]int
		if err := rows.Scan(&got[0], &got[1], &got[2]); err != nil {
			t.Fatal(err)
		}
		if got != want[i] {
			t.Fatalf("history row %d = %v, want %v", i, got, want[i])
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != len(want) {
		t.Fatalf("history row count = %d, want %d", count, len(want))
	}
}
