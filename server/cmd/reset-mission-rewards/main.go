package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
)

const (
	firstMissionID int32 = 3701
	lastMissionID  int32 = 3711
)

type resetStats struct {
	MissionRecords int
	AffectedUsers  int
}

func main() {
	dbPath := flag.String("db", "db/game.db", "SQLite database path")
	dryRun := flag.Bool("dry-run", false, "report changes without writing them")
	flag.Parse()

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	stats, err := resetClaimedMissionRewards(context.Background(), db, *dryRun, time.Now().UnixMilli())
	if err != nil {
		log.Fatalf("reset mission rewards: %v", err)
	}
	if *dryRun {
		log.Printf("dry run: would reset %d claimed mission records for %d users; no changes written",
			stats.MissionRecords, stats.AffectedUsers)
		return
	}
	log.Printf("reset %d claimed mission records for %d users",
		stats.MissionRecords, stats.AffectedUsers)
}

func resetClaimedMissionRewards(ctx context.Context, db *sql.DB, dryRun bool, nowMillis int64) (resetStats, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return resetStats{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stats := resetStats{}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(DISTINCT user_id)
		FROM user_missions
		WHERE mission_id BETWEEN ? AND ? AND mission_progress_status_type=?`,
		firstMissionID, lastMissionID, model.MissionProgressStatusTypeRewardReceived,
	).Scan(&stats.MissionRecords, &stats.AffectedUsers); err != nil {
		return resetStats{}, fmt.Errorf("count claimed mission records: %w", err)
	}
	if dryRun {
		return stats, nil
	}

	result, err := tx.ExecContext(ctx, `UPDATE user_missions
		SET mission_progress_status_type=?, latest_version=?
		WHERE mission_id BETWEEN ? AND ? AND mission_progress_status_type=?`,
		model.MissionProgressStatusTypeClear, nowMillis, firstMissionID, lastMissionID,
		model.MissionProgressStatusTypeRewardReceived)
	if err != nil {
		return resetStats{}, fmt.Errorf("update claimed mission records: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return resetStats{}, fmt.Errorf("count updated mission records: %w", err)
	}
	if updated != int64(stats.MissionRecords) {
		return resetStats{}, fmt.Errorf("updated %d mission records, expected %d", updated, stats.MissionRecords)
	}
	if err := tx.Commit(); err != nil {
		return resetStats{}, fmt.Errorf("commit transaction: %w", err)
	}
	return stats, nil
}
