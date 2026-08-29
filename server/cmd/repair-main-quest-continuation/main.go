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
	replayQuestId       int32
	replayFinalSceneId  int32
	orphanActiveQuestId int32
	apply               bool
}

func main() {
	dbPath := flag.String("db", "db/game.db", "SQLite game database path")
	userId := flag.Int64("user-id", 8, "target user ID")
	stuckQuestId := flag.Int("stuck-quest-id", 210019, "quest referenced by the stale progress scene")
	stuckSceneId := flag.Int("stuck-scene-id", 210019, "stale progress scene ID")
	replayQuestId := flag.Int("replay-quest-id", 0, "replay quest involved in the diagnosed residue")
	replayFinalSceneId := flag.Int("replay-final-scene-id", 0, "final scene identifying a completed replay that should return to the portal")
	orphanActiveQuestId := flag.Int("orphan-active-quest-id", 100021, "unrelated active quest to reset")
	apply := flag.Bool("apply", false, "apply the repair; default is read-only dry-run")
	flag.Parse()
	cfg := repairConfig{
		dbPath:              *dbPath,
		userId:              *userId,
		stuckQuestId:        int32(*stuckQuestId),
		stuckSceneId:        int32(*stuckSceneId),
		replayQuestId:       int32(*replayQuestId),
		replayFinalSceneId:  int32(*replayFinalSceneId),
		orphanActiveQuestId: int32(*orphanActiveQuestId),
		apply:               *apply,
	}

	if err := run(cfg, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "repair failed: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg repairConfig, out io.Writer) error {
	if cfg.userId <= 0 || cfg.stuckQuestId <= 0 || cfg.stuckSceneId <= 0 || cfg.replayQuestId < 0 ||
		cfg.replayFinalSceneId < 0 || cfg.orphanActiveQuestId <= 0 {
		return fmt.Errorf("user and quest identifiers must be positive")
	}
	if cfg.replayFinalSceneId != 0 && cfg.replayQuestId == 0 {
		return fmt.Errorf("replay-final-scene-id requires replay-quest-id")
	}
	if cfg.replayQuestId != 0 &&
		(cfg.replayQuestId == cfg.stuckQuestId || cfg.replayQuestId == cfg.orphanActiveQuestId || cfg.stuckQuestId == cfg.orphanActiveQuestId) {
		return fmt.Errorf("replay, progress, and orphan quest identifiers must be distinct")
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
		printState(out, "rejected", &user, cfg)
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
	if cfg.replayFinalSceneId != 0 {
		return validateCompletedReplayTarget(user, cfg)
	}
	if cfg.replayQuestId != 0 {
		return validateReplayExtraResidueTarget(user, cfg)
	}

	main := user.MainQuest
	if main.CurrentQuestFlowType != int32(model.QuestFlowTypeSubFlow) ||
		main.ProgressQuestFlowType != int32(model.QuestFlowTypeSubFlow) ||
		main.ProgressQuestSceneId != cfg.stuckSceneId ||
		main.ProgressHeadQuestSceneId != cfg.stuckSceneId {
		return fmt.Errorf("user %d main quest progress no longer matches the diagnosed state: flow=%d progressScene=%d progressHead=%d progressFlow=%d", cfg.userId,
			main.CurrentQuestFlowType, main.ProgressQuestSceneId, main.ProgressHeadQuestSceneId, main.ProgressQuestFlowType)
	}

	stuck, ok := user.Quests[cfg.stuckQuestId]
	if !ok || stuck.QuestStateType != model.UserQuestStateTypeCleared || stuck.ClearCount == 0 {
		return fmt.Errorf("quest %d is not the expected cleared quest: present=%v state=%d clear=%d", cfg.stuckQuestId, ok, stuck.QuestStateType, stuck.ClearCount)
	}
	orphan, ok := user.Quests[cfg.orphanActiveQuestId]
	if !ok || orphan.QuestStateType != model.UserQuestStateTypeActive {
		return fmt.Errorf("quest %d is not the expected active residue: present=%v state=%d clear=%d", cfg.orphanActiveQuestId, ok, orphan.QuestStateType, orphan.ClearCount)
	}
	event := user.EventQuest
	if event.CurrentEventQuestChapterId != 0 || event.CurrentQuestId != 0 ||
		event.CurrentQuestSceneId != 0 || event.HeadQuestSceneId != 0 {
		return fmt.Errorf("event quest progress is active; refusing to repair: chapter=%d quest=%d scene=%d head=%d",
			event.CurrentEventQuestChapterId, event.CurrentQuestId, event.CurrentQuestSceneId, event.HeadQuestSceneId)
	}
	return nil
}

func validateCompletedReplayTarget(user *store.UserState, cfg repairConfig) error {
	main := user.MainQuest
	replayFlowType := main.ProgressQuestFlowType
	activeReplay := main.CurrentQuestFlowType == replayFlowType
	partialPortalTransition := main.CurrentQuestFlowType == int32(model.QuestFlowTypeMainFlow) &&
		user.PortalCageStatus.IsCurrentProgress
	replaySceneMatches := main.ReplayFlowCurrentQuestSceneId == cfg.stuckSceneId ||
		main.ReplayFlowCurrentQuestSceneId == cfg.replayFinalSceneId
	if !model.IsReplayQuestFlowType(replayFlowType) ||
		(!activeReplay && !partialPortalTransition) ||
		main.ProgressQuestSceneId != 0 ||
		main.ProgressHeadQuestSceneId != 0 ||
		!replaySceneMatches ||
		main.ReplayFlowHeadQuestSceneId != main.ReplayFlowCurrentQuestSceneId ||
		main.SavedContext.Active ||
		cfg.replayFinalSceneId == cfg.stuckSceneId {
		return fmt.Errorf("user %d completed replay no longer matches the diagnosed state", cfg.userId)
	}

	progress, ok := user.Quests[cfg.stuckQuestId]
	if !ok || progress.QuestStateType != model.UserQuestStateTypeCleared || progress.ClearCount == 0 {
		return fmt.Errorf("progress quest %d is not the expected cleared quest", cfg.stuckQuestId)
	}
	replay, ok := user.Quests[cfg.replayQuestId]
	if !ok || replay.QuestStateType != model.UserQuestStateTypeCleared || replay.ClearCount == 0 {
		return fmt.Errorf("replay quest %d is not the expected cleared quest", cfg.replayQuestId)
	}
	orphan, ok := user.Quests[cfg.orphanActiveQuestId]
	expectedOrphanState := model.UserQuestStateTypeChallenged
	if orphan.ClearCount > 0 {
		expectedOrphanState = model.UserQuestStateTypeCleared
	}
	if !ok || orphan.QuestStateType != expectedOrphanState {
		return fmt.Errorf("quest %d is not in the expected repaired state", cfg.orphanActiveQuestId)
	}
	if eventQuestProgressActive(user) {
		return fmt.Errorf("event quest progress is active; refusing to repair")
	}
	if extraQuestProgressActive(user) {
		return fmt.Errorf("extra quest progress is active; refusing to repair")
	}
	return nil
}

func validateReplayExtraResidueTarget(user *store.UserState, cfg repairConfig) error {
	main := user.MainQuest
	if !model.IsReplayQuestFlowType(main.CurrentQuestFlowType) ||
		main.ProgressQuestFlowType != main.CurrentQuestFlowType ||
		main.ProgressQuestSceneId != cfg.stuckSceneId ||
		main.ProgressHeadQuestSceneId != cfg.stuckSceneId ||
		main.SavedContext.Active {
		return fmt.Errorf("user %d replay progress no longer matches the diagnosed state", cfg.userId)
	}

	progress, ok := user.Quests[cfg.stuckQuestId]
	if !ok || progress.QuestStateType != model.UserQuestStateTypeCleared || progress.ClearCount == 0 {
		return fmt.Errorf("progress quest %d is not the expected cleared quest", cfg.stuckQuestId)
	}
	replay, ok := user.Quests[cfg.replayQuestId]
	if !ok || replay.QuestStateType != model.UserQuestStateTypeActive {
		return fmt.Errorf("replay quest %d is not active", cfg.replayQuestId)
	}
	orphan, ok := user.Quests[cfg.orphanActiveQuestId]
	if !ok || orphan.QuestStateType != model.UserQuestStateTypeActive {
		return fmt.Errorf("quest %d is not the expected active extra-quest residue", cfg.orphanActiveQuestId)
	}
	if eventQuestProgressActive(user) {
		return fmt.Errorf("event quest progress is active; refusing to repair")
	}
	if extraQuestProgressActive(user) {
		return fmt.Errorf("extra quest progress is active; refusing to repair")
	}
	return nil
}

func eventQuestProgressActive(user *store.UserState) bool {
	event := user.EventQuest
	return event.CurrentEventQuestChapterId != 0 || event.CurrentQuestId != 0 ||
		event.CurrentQuestSceneId != 0 || event.HeadQuestSceneId != 0
}

func extraQuestProgressActive(user *store.UserState) bool {
	extra := user.ExtraQuest
	return extra.CurrentQuestId != 0 || extra.CurrentQuestSceneId != 0 || extra.HeadQuestSceneId != 0
}

func applyRepair(user *store.UserState, cfg repairConfig, nowMillis int64) {
	if cfg.replayFinalSceneId != 0 {
		user.MainQuest.ReplayFlowCurrentQuestSceneId = 0
		user.MainQuest.ReplayFlowHeadQuestSceneId = 0
		user.MainQuest.ProgressQuestFlowType = int32(model.QuestFlowTypeUnknown)
		user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeMainFlow)
		user.MainQuest.LatestVersion = nowMillis
		user.PortalCageStatus.IsCurrentProgress = true
		user.PortalCageStatus.LatestVersion = nowMillis
		user.BattleBinary = nil
		user.Battle.LastBattleBinarySize = 0
		return
	}
	if cfg.replayQuestId != 0 {
		orphan := user.Quests[cfg.orphanActiveQuestId]
		if orphan.ClearCount > 0 {
			orphan.QuestStateType = model.UserQuestStateTypeCleared
		} else {
			orphan.QuestStateType = model.UserQuestStateTypeChallenged
		}
		user.Quests[cfg.orphanActiveQuestId] = orphan
		return
	}

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
	if orphan.ClearCount > 0 {
		orphan.QuestStateType = model.UserQuestStateTypeCleared
	} else {
		orphan.QuestStateType = model.UserQuestStateTypeUnknown
	}
	orphan.IsBattleOnly = false
	orphan.UserDeckNumber = 0
	orphan.LatestStartDatetime = 0
	user.Quests[cfg.orphanActiveQuestId] = orphan

	user.BattleBinary = nil
	user.Battle.LastBattleBinarySize = 0
}

func printState(out io.Writer, label string, user *store.UserState, cfg repairConfig) {
	stuck := user.Quests[cfg.stuckQuestId]
	replay := user.Quests[cfg.replayQuestId]
	orphan := user.Quests[cfg.orphanActiveQuestId]
	fmt.Fprintf(out,
		"%s user=%d flow=%d progressScene=%d progressHead=%d progressFlow=%d replayScene=%d replayHead=%d portal=%v stuckQuestState=%d stuckClear=%d replayQuest=%d replayQuestState=%d orphanQuestState=%d orphanClear=%d orphanStart=%d eventQuest=%d eventScene=%d extraQuest=%d extraScene=%d checkpointBytes=%d savedContext=%v\n",
		label, user.UserId, user.MainQuest.CurrentQuestFlowType, user.MainQuest.ProgressQuestSceneId,
		user.MainQuest.ProgressHeadQuestSceneId, user.MainQuest.ProgressQuestFlowType,
		user.MainQuest.ReplayFlowCurrentQuestSceneId, user.MainQuest.ReplayFlowHeadQuestSceneId,
		user.PortalCageStatus.IsCurrentProgress,
		stuck.QuestStateType, stuck.ClearCount, cfg.replayQuestId, replay.QuestStateType, orphan.QuestStateType, orphan.ClearCount, orphan.LatestStartDatetime,
		user.EventQuest.CurrentQuestId, user.EventQuest.CurrentQuestSceneId, user.ExtraQuest.CurrentQuestId, user.ExtraQuest.CurrentQuestSceneId,
		len(user.BattleBinary), user.MainQuest.SavedContext.Active)
}
