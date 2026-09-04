package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"lunar-tear/server/internal/database"
)

const (
	darkFateStoneMaterialID = 315002 // 真暗ノ天命石
	darkFateStoneDeduction  = 630
	supremeAdmirationID     = 300024 // 至高の憧憬
	supremeAdmirationGrant  = 84
)

type adjustmentStats struct {
	Players                      int64
	DarkFateStonesBefore         int64
	DarkFateStonesAfter          int64
	SupremeAdmirationBefore      int64
	SupremeAdmirationAfter       int64
	NegativeDarkFateStonePlayers int64
}

func main() {
	dbPath := flag.String("db", "db/game.db", "SQLite database path")
	apply := flag.Bool("apply", false, "apply the adjustment; default is read-only dry-run")
	flag.Parse()

	if _, err := os.Stat(*dbPath); err != nil {
		log.Fatalf("stat database %q: %v", *dbPath, err)
	}
	db, err := openAdjustmentDatabase(*dbPath, *apply)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	stats, err := adjustAllPlayerMaterials(context.Background(), db, *apply)
	if err != nil {
		log.Fatalf("adjust player materials: %v", err)
	}
	mode := "dry run"
	if *apply {
		mode = "applied"
	}
	log.Printf("%s for %d players: 真暗ノ天命石 %d -> %d (%d players negative); 至高の憧憬 %d -> %d",
		mode, stats.Players, stats.DarkFateStonesBefore, stats.DarkFateStonesAfter,
		stats.NegativeDarkFateStonePlayers, stats.SupremeAdmirationBefore, stats.SupremeAdmirationAfter)
	if !*apply {
		log.Print("no changes written; stop the game server and rerun with --apply to commit this one-time adjustment")
	}
}

func openAdjustmentDatabase(path string, apply bool) (*sql.DB, error) {
	if apply {
		return database.Open(path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(absPath)+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open database read-only: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database read-only: %w", err)
	}
	return db, nil
}

func adjustAllPlayerMaterials(ctx context.Context, db *sql.DB, apply bool) (adjustmentStats, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return adjustmentStats{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stats, err := loadAdjustmentStats(ctx, tx)
	if err != nil {
		return adjustmentStats{}, err
	}
	stats.DarkFateStonesAfter = stats.DarkFateStonesBefore - stats.Players*darkFateStoneDeduction
	stats.SupremeAdmirationAfter = stats.SupremeAdmirationBefore + stats.Players*supremeAdmirationGrant
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM users AS u
		LEFT JOIN user_materials AS m
			ON m.user_id=u.user_id AND m.material_id=?
		WHERE COALESCE(m.count, 0) - ? < 0`, darkFateStoneMaterialID, darkFateStoneDeduction).
		Scan(&stats.NegativeDarkFateStonePlayers); err != nil {
		return adjustmentStats{}, fmt.Errorf("count resulting negative inventories: %w", err)
	}
	if !apply {
		return stats, nil
	}

	if err := upsertMaterialDelta(ctx, tx, darkFateStoneMaterialID, -darkFateStoneDeduction, stats.Players); err != nil {
		return adjustmentStats{}, fmt.Errorf("deduct 真暗ノ天命石: %w", err)
	}
	if err := upsertMaterialDelta(ctx, tx, supremeAdmirationID, supremeAdmirationGrant, stats.Players); err != nil {
		return adjustmentStats{}, fmt.Errorf("grant 至高の憧憬: %w", err)
	}
	if err := verifyAdjustment(ctx, tx, stats); err != nil {
		return adjustmentStats{}, err
	}
	if err := tx.Commit(); err != nil {
		return adjustmentStats{}, fmt.Errorf("commit transaction: %w", err)
	}
	return stats, nil
}

func loadAdjustmentStats(ctx context.Context, tx *sql.Tx) (adjustmentStats, error) {
	var stats adjustmentStats
	err := tx.QueryRowContext(ctx, `SELECT
		COUNT(DISTINCT u.user_id),
		COALESCE(SUM(CASE WHEN m.material_id=? THEN m.count ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN m.material_id=? THEN m.count ELSE 0 END), 0)
		FROM users AS u
		LEFT JOIN user_materials AS m
			ON m.user_id=u.user_id AND m.material_id IN (?, ?)`,
		darkFateStoneMaterialID, supremeAdmirationID,
		darkFateStoneMaterialID, supremeAdmirationID,
	).Scan(&stats.Players, &stats.DarkFateStonesBefore, &stats.SupremeAdmirationBefore)
	if err != nil {
		return adjustmentStats{}, fmt.Errorf("load adjustment totals: %w", err)
	}
	return stats, nil
}

func upsertMaterialDelta(ctx context.Context, tx *sql.Tx, materialID, delta, expectedPlayers int64) error {
	result, err := tx.ExecContext(ctx, `INSERT INTO user_materials (user_id, material_id, count)
		SELECT user_id, ?, ?
		FROM users
		WHERE TRUE
		ON CONFLICT(user_id, material_id) DO UPDATE
		SET count=user_materials.count + excluded.count`, materialID, delta)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count changed rows: %w", err)
	}
	if changed != expectedPlayers {
		return fmt.Errorf("changed %d rows for material %d, expected %d", changed, materialID, expectedPlayers)
	}
	return nil
}

func verifyAdjustment(ctx context.Context, tx *sql.Tx, expected adjustmentStats) error {
	actual, err := loadAdjustmentStats(ctx, tx)
	if err != nil {
		return fmt.Errorf("verify adjustment: %w", err)
	}
	if actual.Players != expected.Players ||
		actual.DarkFateStonesBefore != expected.DarkFateStonesAfter ||
		actual.SupremeAdmirationBefore != expected.SupremeAdmirationAfter {
		return fmt.Errorf("verification failed: players=%d dark=%d admiration=%d, expected %d/%d/%d",
			actual.Players, actual.DarkFateStonesBefore, actual.SupremeAdmirationBefore,
			expected.Players, expected.DarkFateStonesAfter, expected.SupremeAdmirationAfter)
	}
	return nil
}
