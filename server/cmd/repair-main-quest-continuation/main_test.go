package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestRepairDiagnosedMainQuestContinuationState(t *testing.T) {
	cfg := repairConfig{
		userId:              8,
		stuckQuestId:        210019,
		stuckSceneId:        210019,
		orphanActiveQuestId: 100021,
	}
	user := diagnosedUserState(cfg)
	user.BattleBinary = []byte("stale checkpoint")
	user.Battle.LastBattleBinarySize = int32(len(user.BattleBinary))

	if err := validateRepairTarget(user, cfg); err != nil {
		t.Fatalf("validate diagnosed state: %v", err)
	}
	applyRepair(user, cfg, 123)

	if user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeUnknown) ||
		user.MainQuest.ProgressQuestSceneId != 0 ||
		user.MainQuest.ProgressHeadQuestSceneId != 0 ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeUnknown) {
		t.Fatalf("main quest progress was not cleared: %+v", user.MainQuest)
	}
	orphan := user.Quests[cfg.orphanActiveQuestId]
	if orphan.QuestStateType != model.UserQuestStateTypeUnknown || orphan.LatestStartDatetime != 0 {
		t.Fatalf("orphan active quest was not reset: %+v", orphan)
	}
	if len(user.BattleBinary) != 0 || user.Battle.LastBattleBinarySize != 0 {
		t.Fatal("stale battle checkpoint was not cleared")
	}
}

func TestRepairRefusesChangedAccountState(t *testing.T) {
	cfg := repairConfig{
		userId:              8,
		stuckQuestId:        210019,
		stuckSceneId:        210019,
		orphanActiveQuestId: 100021,
	}
	user := diagnosedUserState(cfg)
	user.EventQuest.CurrentQuestId = 100021

	if err := validateRepairTarget(user, cfg); err == nil {
		t.Fatal("repair accepted an account with active event quest progress")
	}
}

func TestRepairRestoresSavedMainQuestContext(t *testing.T) {
	cfg := repairConfig{
		userId:              8,
		stuckQuestId:        210019,
		stuckSceneId:        210019,
		orphanActiveQuestId: 100021,
	}
	user := diagnosedUserState(cfg)
	user.MainQuest.SavedContext = store.SavedQuestContext{
		Active:                  true,
		CurrentQuestSceneId:     548,
		HeadQuestSceneId:        548,
		CurrentMainQuestRouteId: 1,
		MainQuestSeasonId:       2,
		CurrentQuestFlowType:    int32(model.QuestFlowTypeMainFlow),
		PortalCageInProgress:    true,
	}

	applyRepair(user, cfg, 123)

	if user.MainQuest.SavedContext.Active || user.MainQuest.CurrentQuestSceneId != 548 ||
		user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeMainFlow) ||
		!user.PortalCageStatus.IsCurrentProgress {
		t.Fatalf("saved main quest context was not restored: %+v", user.MainQuest)
	}
}

func TestRunDryRunThenApplyAgainstSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "game.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("repair-test", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	cfg := repairConfig{
		dbPath:              dbPath,
		userId:              userId,
		stuckQuestId:        210019,
		stuckSceneId:        210019,
		orphanActiveQuestId: 100021,
	}
	diagnosed := diagnosedUserState(cfg)
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.MainQuest = diagnosed.MainQuest
		user.EventQuest = diagnosed.EventQuest
		user.Quests[cfg.stuckQuestId] = diagnosed.Quests[cfg.stuckQuestId]
		user.Quests[cfg.orphanActiveQuestId] = diagnosed.Quests[cfg.orphanActiveQuestId]
		user.BattleBinary = []byte("stale checkpoint")
		user.Battle.LastBattleBinarySize = int32(len(user.BattleBinary))
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := run(cfg, &output); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !bytes.Contains(output.Bytes(), []byte("dry-run only")) {
		t.Fatalf("dry-run output = %q", output.String())
	}

	cfg.apply = true
	output.Reset()
	if err := run(cfg, &output); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !bytes.Contains(output.Bytes(), []byte("repaired user=")) {
		t.Fatalf("apply output = %q", output.String())
	}

	db, err = database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repaired, err := sqlite.New(db, nil).LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.MainQuest.ProgressQuestSceneId != 0 ||
		repaired.Quests[cfg.orphanActiveQuestId].QuestStateType != model.UserQuestStateTypeUnknown ||
		len(repaired.BattleBinary) != 0 {
		t.Fatal("repair was not persisted")
	}
}

func diagnosedUserState(cfg repairConfig) *store.UserState {
	user := store.SeedUserState(cfg.userId, "test", 1, model.ClientPlatform{})
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeSubFlow)
	user.MainQuest.ProgressQuestFlowType = int32(model.QuestFlowTypeSubFlow)
	user.MainQuest.ProgressQuestSceneId = cfg.stuckSceneId
	user.MainQuest.ProgressHeadQuestSceneId = cfg.stuckSceneId
	user.Quests[cfg.stuckQuestId] = store.UserQuestState{
		QuestId:        cfg.stuckQuestId,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     1,
	}
	user.Quests[cfg.orphanActiveQuestId] = store.UserQuestState{
		QuestId:             cfg.orphanActiveQuestId,
		QuestStateType:      model.UserQuestStateTypeActive,
		LatestStartDatetime: 10,
	}
	return user
}
