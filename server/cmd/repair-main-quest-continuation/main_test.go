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

func TestRepairReplayExtraResiduePreservesMainQuestContinuation(t *testing.T) {
	cfg := repairConfig{
		userId:              2,
		stuckQuestId:        334,
		stuckSceneId:        845,
		replayQuestId:       30330,
		orphanActiveQuestId: 381,
	}
	user := diagnosedReplayExtraResidueState(cfg)
	user.BattleBinary = []byte("replay checkpoint")
	user.Battle.LastBattleBinarySize = int32(len(user.BattleBinary))

	if err := validateRepairTarget(user, cfg); err != nil {
		t.Fatalf("validate replay residue: %v", err)
	}
	applyRepair(user, cfg, 123)

	if user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeReplayFlow) ||
		user.MainQuest.ProgressQuestSceneId != cfg.stuckSceneId ||
		user.MainQuest.ProgressHeadQuestSceneId != cfg.stuckSceneId ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeReplayFlow) {
		t.Fatalf("replay continuation was changed: %+v", user.MainQuest)
	}
	if got := user.Quests[cfg.replayQuestId].QuestStateType; got != model.UserQuestStateTypeActive {
		t.Fatalf("replay quest state = %d, want active", got)
	}
	if got := user.Quests[cfg.orphanActiveQuestId].QuestStateType; got != model.UserQuestStateTypeChallenged {
		t.Fatalf("orphan extra quest state = %d, want challenged", got)
	}
	if string(user.BattleBinary) != "replay checkpoint" || user.Battle.LastBattleBinarySize != int32(len(user.BattleBinary)) {
		t.Fatal("replay battle checkpoint was changed")
	}
}

func TestRepairReplayExtraResidueRestoresPreviouslyClearedQuest(t *testing.T) {
	cfg := repairConfig{
		userId:              2,
		stuckQuestId:        334,
		stuckSceneId:        845,
		replayQuestId:       30330,
		orphanActiveQuestId: 381,
	}
	user := diagnosedReplayExtraResidueState(cfg)
	orphan := user.Quests[cfg.orphanActiveQuestId]
	orphan.ClearCount = 1
	user.Quests[cfg.orphanActiveQuestId] = orphan

	if err := validateRepairTarget(user, cfg); err != nil {
		t.Fatalf("validate cleared replay residue: %v", err)
	}
	applyRepair(user, cfg, 123)

	if got := user.Quests[cfg.orphanActiveQuestId].QuestStateType; got != model.UserQuestStateTypeCleared {
		t.Fatalf("orphan extra quest state = %d, want cleared", got)
	}
}

func TestRepairReplayExtraResidueRefusesActiveExtraQuestProgress(t *testing.T) {
	cfg := repairConfig{
		userId:              2,
		stuckQuestId:        334,
		stuckSceneId:        845,
		replayQuestId:       30330,
		orphanActiveQuestId: 381,
	}
	user := diagnosedReplayExtraResidueState(cfg)
	user.ExtraQuest.CurrentQuestId = cfg.orphanActiveQuestId

	if err := validateRepairTarget(user, cfg); err == nil {
		t.Fatal("repair accepted active extra quest progress")
	}
}

func TestRepairCompletedReplayExitsToPortal(t *testing.T) {
	cfg := repairConfig{
		userId:              2,
		stuckQuestId:        334,
		stuckSceneId:        845,
		replayQuestId:       30330,
		replayFinalSceneId:  848,
		orphanActiveQuestId: 381,
	}
	user := diagnosedCompletedReplayState(cfg)
	user.BattleBinary = []byte("stale replay checkpoint")
	user.Battle.LastBattleBinarySize = int32(len(user.BattleBinary))

	if err := validateRepairTarget(user, cfg); err != nil {
		t.Fatalf("validate completed replay: %v", err)
	}
	applyRepair(user, cfg, 123)

	if user.MainQuest.ReplayFlowCurrentQuestSceneId != cfg.replayFinalSceneId ||
		user.MainQuest.ReplayFlowHeadQuestSceneId != cfg.replayFinalSceneId ||
		user.MainQuest.LatestVersion != 123 {
		t.Fatalf("completed replay was not advanced: %+v", user.MainQuest)
	}
	if user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeMainFlow) ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeReplayFlow) ||
		!user.PortalCageStatus.IsCurrentProgress || user.PortalCageStatus.LatestVersion != 123 {
		t.Fatalf("completed replay did not exit to portal: %+v", user.MainQuest)
	}
	if got := user.Quests[cfg.replayQuestId].QuestStateType; got != model.UserQuestStateTypeCleared {
		t.Fatalf("replay quest state = %d, want cleared", got)
	}
	if len(user.BattleBinary) != 0 || user.Battle.LastBattleBinarySize != 0 {
		t.Fatal("stale replay checkpoint was not cleared")
	}
}

func TestRepairCompletedReplayAcceptsAlreadyAdvancedEndingScene(t *testing.T) {
	cfg := repairConfig{
		userId:              2,
		stuckQuestId:        334,
		stuckSceneId:        845,
		replayQuestId:       30330,
		replayFinalSceneId:  848,
		orphanActiveQuestId: 381,
	}
	user := diagnosedCompletedReplayState(cfg)
	user.MainQuest.ReplayFlowCurrentQuestSceneId = cfg.replayFinalSceneId
	user.MainQuest.ReplayFlowHeadQuestSceneId = cfg.replayFinalSceneId

	if err := validateRepairTarget(user, cfg); err != nil {
		t.Fatalf("validate already advanced replay: %v", err)
	}
	applyRepair(user, cfg, 123)

	if user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeMainFlow) ||
		!user.PortalCageStatus.IsCurrentProgress {
		t.Fatalf("already advanced replay did not exit to portal: %+v", user.MainQuest)
	}
}

func TestRepairCompletedReplayRefusesActiveReplayQuest(t *testing.T) {
	cfg := repairConfig{
		userId:              2,
		stuckQuestId:        334,
		stuckSceneId:        845,
		replayQuestId:       30330,
		replayFinalSceneId:  848,
		orphanActiveQuestId: 381,
	}
	user := diagnosedCompletedReplayState(cfg)
	replay := user.Quests[cfg.replayQuestId]
	replay.QuestStateType = model.UserQuestStateTypeActive
	user.Quests[cfg.replayQuestId] = replay

	if err := validateRepairTarget(user, cfg); err == nil {
		t.Fatal("completed replay repair accepted an active replay quest")
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

func TestRunReplayExtraResidueDryRunThenApplyAgainstSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "game.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("replay-repair-test", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	cfg := repairConfig{
		dbPath:              dbPath,
		userId:              userId,
		stuckQuestId:        334,
		stuckSceneId:        845,
		replayQuestId:       30330,
		orphanActiveQuestId: 381,
	}
	diagnosed := diagnosedReplayExtraResidueState(cfg)
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.MainQuest = diagnosed.MainQuest
		user.EventQuest = diagnosed.EventQuest
		user.ExtraQuest = diagnosed.ExtraQuest
		user.Quests[cfg.stuckQuestId] = diagnosed.Quests[cfg.stuckQuestId]
		user.Quests[cfg.replayQuestId] = diagnosed.Quests[cfg.replayQuestId]
		user.Quests[cfg.orphanActiveQuestId] = diagnosed.Quests[cfg.orphanActiveQuestId]
		user.BattleBinary = []byte("replay checkpoint")
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

	db, err = database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repaired, err := sqlite.New(db, nil).LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.Quests[cfg.orphanActiveQuestId].QuestStateType != model.UserQuestStateTypeChallenged {
		t.Fatal("extra-quest residue was not repaired")
	}
	if repaired.MainQuest.ProgressQuestSceneId != cfg.stuckSceneId ||
		repaired.Quests[cfg.replayQuestId].QuestStateType != model.UserQuestStateTypeActive ||
		string(repaired.BattleBinary) != "replay checkpoint" {
		t.Fatal("replay continuation was not preserved")
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

func diagnosedReplayExtraResidueState(cfg repairConfig) *store.UserState {
	user := store.SeedUserState(cfg.userId, "test", 1, model.ClientPlatform{})
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ProgressQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ProgressQuestSceneId = cfg.stuckSceneId
	user.MainQuest.ProgressHeadQuestSceneId = cfg.stuckSceneId
	user.Quests[cfg.stuckQuestId] = store.UserQuestState{
		QuestId:        cfg.stuckQuestId,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     1,
	}
	user.Quests[cfg.replayQuestId] = store.UserQuestState{
		QuestId:        cfg.replayQuestId,
		QuestStateType: model.UserQuestStateTypeActive,
	}
	user.Quests[cfg.orphanActiveQuestId] = store.UserQuestState{
		QuestId:             cfg.orphanActiveQuestId,
		QuestStateType:      model.UserQuestStateTypeActive,
		LatestStartDatetime: 10,
	}
	return user
}

func diagnosedCompletedReplayState(cfg repairConfig) *store.UserState {
	user := store.SeedUserState(cfg.userId, "test", 1, model.ClientPlatform{})
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ProgressQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ReplayFlowCurrentQuestSceneId = cfg.stuckSceneId
	user.MainQuest.ReplayFlowHeadQuestSceneId = cfg.stuckSceneId
	user.Quests[cfg.stuckQuestId] = store.UserQuestState{
		QuestId:        cfg.stuckQuestId,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     1,
	}
	user.Quests[cfg.replayQuestId] = store.UserQuestState{
		QuestId:        cfg.replayQuestId,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     1,
	}
	user.Quests[cfg.orphanActiveQuestId] = store.UserQuestState{
		QuestId:        cfg.orphanActiveQuestId,
		QuestStateType: model.UserQuestStateTypeChallenged,
	}
	return user
}
