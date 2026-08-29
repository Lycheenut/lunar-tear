package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestHandleQuestRestartRecoversClearedMismatchedProgress(t *testing.T) {
	h, user := mismatchedContinuationFixture()

	if err := h.HandleQuestRestart(user, 210031, 123); err != nil {
		t.Fatalf("restart recoverable continuation: %v", err)
	}
	if user.MainQuest.ProgressQuestSceneId != 210031 ||
		user.MainQuest.ProgressHeadQuestSceneId != 210031 ||
		user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeSubFlow) ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeSubFlow) ||
		user.MainQuest.LatestVersion != 123 {
		t.Fatalf("continuation was not rebound: %+v", user.MainQuest)
	}
	if got := user.Quests[210031]; got.QuestStateType != model.UserQuestStateTypeActive || got.LatestStartDatetime != 123 {
		t.Fatalf("active quest was not restarted: %+v", got)
	}
}

func TestHandleQuestRestartDoesNotReplaceAnotherActiveProgress(t *testing.T) {
	h, user := mismatchedContinuationFixture()
	progress := user.Quests[10014]
	progress.QuestStateType = model.UserQuestStateTypeActive
	user.Quests[10014] = progress

	if err := h.HandleQuestRestart(user, 210031, 123); err == nil {
		t.Fatal("restart replaced another active quest progress")
	}
	if user.MainQuest.ProgressQuestSceneId != 10014 || user.Quests[210031].LatestStartDatetime != 10 {
		t.Fatal("rejected restart mutated continuation state")
	}
}

func TestHandleQuestRestartRecoversNonActiveProgressWithoutClearHistory(t *testing.T) {
	h, user := mismatchedContinuationFixture()
	user.Quests[10014] = store.UserQuestState{
		QuestId:        10014,
		QuestStateType: model.UserQuestStateTypeChallenged,
	}

	if err := h.HandleQuestRestart(user, 210031, 123); err != nil {
		t.Fatalf("restart non-active continuation: %v", err)
	}
	if user.MainQuest.ProgressQuestSceneId != 210031 {
		t.Fatalf("non-active continuation was not rebound: %+v", user.MainQuest)
	}
}

func TestHandleStaleMainQuestRetireClearsOnlyProgress(t *testing.T) {
	h, user := mismatchedContinuationFixture()
	user.BattleBinary = []byte("stale")

	if !h.HandleStaleMainQuestRetire(user, 10014, 123) {
		t.Fatal("stale cleared progress was not retired")
	}
	if user.MainQuest.ProgressQuestSceneId != 0 ||
		user.MainQuest.ProgressHeadQuestSceneId != 0 ||
		user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeUnknown) ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeUnknown) ||
		user.MainQuest.LatestVersion != 123 {
		t.Fatalf("stale progress was not cleared: %+v", user.MainQuest)
	}
	if len(user.BattleBinary) != 0 {
		t.Fatal("stale battle checkpoint was not cleared")
	}
	if got := user.Quests[10014]; got.QuestStateType != model.UserQuestStateTypeCleared || got.ClearCount != 1 {
		t.Fatalf("cleared quest was changed: %+v", got)
	}
	if got := user.Quests[210031]; got.QuestStateType != model.UserQuestStateTypeActive || got.LatestStartDatetime != 10 {
		t.Fatalf("unrelated active quest was changed: %+v", got)
	}
}

func TestHandleStaleMainQuestRetireExitsReplayFlow(t *testing.T) {
	h, user := mismatchedContinuationFixture()
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ProgressQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ReplayFlowCurrentQuestSceneId = 10014
	user.MainQuest.ReplayFlowHeadQuestSceneId = 10014
	user.Quests[10014] = store.UserQuestState{
		QuestId:        10014,
		QuestStateType: model.UserQuestStateTypeChallenged,
	}

	if !h.HandleStaleMainQuestRetire(user, 10014, 123) {
		t.Fatal("stale replay-flow progress was not retired")
	}
	if user.MainQuest.ProgressQuestSceneId != 0 ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeUnknown) ||
		user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeMainFlow) ||
		user.MainQuest.ReplayFlowCurrentQuestSceneId != 0 ||
		user.MainQuest.ReplayFlowHeadQuestSceneId != 0 ||
		!user.PortalCageStatus.IsCurrentProgress {
		t.Fatalf("stale replay flow was not exited: main=%+v portal=%+v", user.MainQuest, user.PortalCageStatus)
	}
}

func TestHandleStaleMainQuestRetireClearsNonActiveProgressWithoutClearHistory(t *testing.T) {
	h, user := mismatchedContinuationFixture()
	user.Quests[10014] = store.UserQuestState{
		QuestId:        10014,
		QuestStateType: model.UserQuestStateTypeUnknown,
	}

	if !h.HandleStaleMainQuestRetire(user, 10014, 123) {
		t.Fatal("non-active progress without clear history was not retired")
	}
	if user.MainQuest.ProgressQuestSceneId != 0 {
		t.Fatalf("non-active progress was not cleared: %+v", user.MainQuest)
	}
	if got := user.Quests[10014]; got.QuestStateType != model.UserQuestStateTypeUnknown {
		t.Fatalf("quest state was changed: %+v", got)
	}
}

func TestHandleStaleMainQuestRetireRestoresSavedContext(t *testing.T) {
	h, user := mismatchedContinuationFixture()
	user.Quests[10014] = store.UserQuestState{
		QuestId:        10014,
		QuestStateType: model.UserQuestStateTypeChallenged,
	}
	user.MainQuest.SavedContext = store.SavedQuestContext{
		Active:                  true,
		CurrentQuestSceneId:     548,
		HeadQuestSceneId:        550,
		CurrentMainQuestRouteId: 1,
		MainQuestSeasonId:       2,
		CurrentQuestFlowType:    int32(model.QuestFlowTypeMainFlow),
		PortalCageInProgress:    true,
	}

	if !h.HandleStaleMainQuestRetire(user, 10014, 123) {
		t.Fatal("stale menu replay progress was not retired")
	}
	if user.MainQuest.SavedContext.Active ||
		user.MainQuest.CurrentQuestSceneId != 548 ||
		user.MainQuest.HeadQuestSceneId != 550 ||
		user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeMainFlow) ||
		!user.PortalCageStatus.IsCurrentProgress {
		t.Fatalf("saved context was not restored: main=%+v portal=%+v", user.MainQuest, user.PortalCageStatus)
	}
}

func mismatchedContinuationFixture() (*QuestHandler, *store.UserState) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			10014:  {QuestId: 10014, IsCountedAsQuest: true},
			210031: {QuestId: 210031, IsCountedAsQuest: true},
		},
		SceneById: map[int32]masterdata.EntityMQuestScene{
			10014:  {QuestSceneId: 10014, QuestId: 10014},
			210031: {QuestSceneId: 210031, QuestId: 210031},
		},
		SceneIdsByQuestId: map[int32][]int32{
			10014:  {10014},
			210031: {210031},
		},
	}}
	user := store.SeedUserState(8, "test", 1, model.ClientPlatform{})
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeSubFlow)
	user.MainQuest.ProgressQuestFlowType = int32(model.QuestFlowTypeSubFlow)
	user.MainQuest.ProgressQuestSceneId = 10014
	user.MainQuest.ProgressHeadQuestSceneId = 10014
	user.Quests[10014] = store.UserQuestState{
		QuestId:        10014,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     1,
	}
	user.Quests[210031] = store.UserQuestState{
		QuestId:             210031,
		QuestStateType:      model.UserQuestStateTypeActive,
		LatestStartDatetime: 10,
	}
	return h, user
}
