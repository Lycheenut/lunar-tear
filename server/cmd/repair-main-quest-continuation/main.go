package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
)

type repairConfig struct {
	dbPath              string
	userId              int64
	stuckQuestId        int32
	stuckSceneId        int32
	orphanActiveQuestId int32
	apply               bool
}

func main() {
	dbPath := flag.String("db", "db/game.db", "SQLite game database path")
	userId := flag.Int64("user-id", 8, "target user ID")
	stuckQuestId := flag.Int("stuck-quest-id", 210019, "quest referenced by the stale progress scene")
	stuckSceneId := flag.Int("stuck-scene-id", 210019, "stale progress scene ID")
	orphanActiveQuestId := flag.Int("orphan-active-quest-id", 100021, "unrelated active quest to reset")
	apply := flag.Bool("apply", false, "apply the repair; default is read-only dry-run")
	flag.Parse()
	cfg := repairConfig{
		dbPath:              *dbPath,
		userId:              *userId,
		stuckQuestId:        int32(*stuckQuestId),
		stuckSceneId:        int32(*stuckSceneId),
		orphanActiveQuestId: int32(*orphanActiveQuestId),
		apply:               *apply,
	}

	if err := run(cfg, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "repair failed: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg repairConfig, out io.Writer) error {
	if cfg.userId <= 0 || cfg.stuckQuestId <= 0 || cfg.stuckSceneId <= 0 || cfg.orphanActiveQuestId <= 0 {
		return fmt.Errorf("user and quest identifiers must be positive")
	}
	if _, err := os.Stat(cfg.dbPath); err != nil {
		return fmt.Errorf("stat database %q: %w", cfg.dbPath, err)
	}

	db, err := openDatabase(cfg.dbPath, !cfg.apply)
	if err != nil {
		return err
	}
	defer db.Close()

	users := sqlite.New(db, nil)
	user, err := users.LoadUser(cfg.userId)
	if err != nil {
		return fmt.Errorf("load user %d: %w", cfg.userId, err)
	}
	if err := validateRepairTarget(&user, cfg); err != nil {
		return err
	}
	printState(out, "matched", &user, cfg)

	if !cfg.apply {
		fmt.Fprintln(out, "dry-run only; rerun with --apply to commit this exact repair")
		return nil
	}

	updated, err := users.UpdateUsers([]int64{cfg.userId}, func(locked map[int64]*store.UserState) error {
		user := locked[cfg.userId]
		if err := validateRepairTarget(user, cfg); err != nil {
			return err
		}
		applyRepair(user, cfg, time.Now().UnixMilli())
		return nil
	})
	if err != nil {
		return fmt.Errorf("update user %d: %w", cfg.userId, err)
	}
	repaired := updated[cfg.userId]
	printState(out, "repaired", &repaired, cfg)
	return nil
}

func openDatabase(path string, readOnly bool) (*sql.DB, error) {
	if !readOnly {
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

func validateRepairTarget(user *store.UserState, cfg repairConfig) error {
	main := user.MainQuest
	if main.CurrentQuestFlowType != int32(model.QuestFlowTypeSubFlow) ||
		main.ProgressQuestFlowType != int32(model.QuestFlowTypeSubFlow) ||
		main.ProgressQuestSceneId != cfg.stuckSceneId ||
		main.ProgressHeadQuestSceneId != cfg.stuckSceneId {
		return fmt.Errorf("user %d main quest progress no longer matches the diagnosed state", cfg.userId)
	}

	stuck, ok := user.Quests[cfg.stuckQuestId]
	if !ok || stuck.QuestStateType != model.UserQuestStateTypeCleared || stuck.ClearCount == 0 {
		return fmt.Errorf("quest %d is not the expected cleared quest", cfg.stuckQuestId)
	}
	orphan, ok := user.Quests[cfg.orphanActiveQuestId]
	if !ok || orphan.QuestStateType != model.UserQuestStateTypeActive || orphan.ClearCount != 0 {
		return fmt.Errorf("quest %d is not the expected uncleared active residue", cfg.orphanActiveQuestId)
	}
	event := user.EventQuest
	if event.CurrentEventQuestChapterId != 0 || event.CurrentQuestId != 0 ||
		event.CurrentQuestSceneId != 0 || event.HeadQuestSceneId != 0 {
		return fmt.Errorf("event quest progress is active; refusing to repair")
	}
	return nil
}

func applyRepair(user *store.UserState, cfg repairConfig, nowMillis int64) {
	main := &user.MainQuest
	main.ProgressQuestSceneId = 0
	main.ProgressHeadQuestSceneId = 0
	main.ProgressQuestFlowType = int32(model.QuestFlowTypeUnknown)
	main.CurrentQuestFlowType = int32(model.QuestFlowTypeUnknown)

	if main.SavedContext.Active {
		ctx := main.SavedContext
		main.CurrentQuestSceneId = ctx.CurrentQuestSceneId
		main.HeadQuestSceneId = ctx.HeadQuestSceneId
		main.CurrentMainQuestRouteId = ctx.CurrentMainQuestRouteId
		main.MainQuestSeasonId = ctx.MainQuestSeasonId
		main.IsReachedLastQuestScene = ctx.IsReachedLastQuestScene
		main.CurrentQuestFlowType = ctx.CurrentQuestFlowType
		user.PortalCageStatus.IsCurrentProgress = ctx.PortalCageInProgress
		user.PortalCageStatus.LatestVersion = nowMillis
		main.SavedContext = store.SavedQuestContext{}
	}
	main.LatestVersion = nowMillis

	orphan := user.Quests[cfg.orphanActiveQuestId]
	orphan.QuestStateType = model.UserQuestStateTypeUnknown
	orphan.IsBattleOnly = false
	orphan.UserDeckNumber = 0
	orphan.LatestStartDatetime = 0
	user.Quests[cfg.orphanActiveQuestId] = orphan

	user.BattleBinary = nil
	user.Battle.LastBattleBinarySize = 0
}

func printState(out io.Writer, label string, user *store.UserState, cfg repairConfig) {
	stuck := user.Quests[cfg.stuckQuestId]
	orphan := user.Quests[cfg.orphanActiveQuestId]
	fmt.Fprintf(out,
		"%s user=%d flow=%d progressScene=%d progressHead=%d progressFlow=%d stuckQuestState=%d orphanQuestState=%d orphanStart=%d checkpointBytes=%d savedContext=%v\n",
		label, user.UserId, user.MainQuest.CurrentQuestFlowType, user.MainQuest.ProgressQuestSceneId,
		user.MainQuest.ProgressHeadQuestSceneId, user.MainQuest.ProgressQuestFlowType,
		stuck.QuestStateType, orphan.QuestStateType, orphan.LatestStartDatetime,
		len(user.BattleBinary), user.MainQuest.SavedContext.Active)
}
