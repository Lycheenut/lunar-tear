package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestHandleMainQuestSceneProgressIgnoresStaleClearedQuest(t *testing.T) {
	const (
		staleQuestId  int32 = 10014
		staleSceneId  int32 = 10014
		activeQuestId int32 = 210031
	)
	h := mainQuestSceneProgressFixture(staleQuestId, staleSceneId)
	user := store.SeedUserState(8, "test", 1, model.ClientPlatform{})
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeMainFlow)
	user.Quests[staleQuestId] = store.UserQuestState{
		QuestId:        staleQuestId,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     1,
	}
	user.Quests[activeQuestId] = store.UserQuestState{
		QuestId:        activeQuestId,
		QuestStateType: model.UserQuestStateTypeActive,
	}

	if err := h.HandleMainQuestSceneProgress(user, staleSceneId); err != nil {
		t.Fatalf("stale cleared scene update: %v", err)
	}
	if user.MainQuest.ProgressQuestSceneId != 0 ||
		user.MainQuest.ProgressHeadQuestSceneId != 0 ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeUnknown) ||
		user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeMainFlow) {
		t.Fatalf("stale cleared scene changed main quest progress: %+v", user.MainQuest)
	}
}

func TestHandleMainQuestSceneProgressAllowsActiveQuest(t *testing.T) {
	const (
		questId int32 = 13
		sceneId int32 = 22
	)
	h := mainQuestSceneProgressFixture(questId, sceneId)
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[questId] = store.UserQuestState{
		QuestId:        questId,
		QuestStateType: model.UserQuestStateTypeActive,
	}

	if err := h.HandleMainQuestSceneProgress(user, sceneId); err != nil {
		t.Fatalf("active scene update: %v", err)
	}
	if user.MainQuest.ProgressQuestSceneId != sceneId ||
		user.MainQuest.ProgressHeadQuestSceneId != sceneId ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeSubFlow) {
		t.Fatalf("active scene was not recorded: %+v", user.MainQuest)
	}
}

func TestHandleMainQuestSceneProgressAllowsClearedMenuReplay(t *testing.T) {
	const (
		questId int32 = 13
		sceneId int32 = 22
	)
	h := mainQuestSceneProgressFixture(questId, sceneId)
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[questId] = store.UserQuestState{
		QuestId:        questId,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     1,
	}
	user.MainQuest.SavedContext.Active = true
	user.MainQuest.ProgressQuestSceneId = sceneId

	if err := h.HandleMainQuestSceneProgress(user, sceneId); err != nil {
		t.Fatalf("cleared menu replay scene update: %v", err)
	}
	if user.MainQuest.ProgressQuestSceneId != sceneId ||
		user.MainQuest.ProgressHeadQuestSceneId != sceneId ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeSubFlow) {
		t.Fatalf("menu replay scene was not recorded: %+v", user.MainQuest)
	}
}

func TestHandleMainQuestSceneProgressAllowsReplayFlowScene(t *testing.T) {
	const (
		questId int32 = 13
		sceneId int32 = 22
	)
	h := mainQuestSceneProgressFixture(questId, sceneId)
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[questId] = store.UserQuestState{
		QuestId:        questId,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     1,
	}
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeReplayFlow)

	if err := h.HandleMainQuestSceneProgress(user, sceneId); err != nil {
		t.Fatalf("replay-flow scene update: %v", err)
	}
	if user.MainQuest.ProgressQuestSceneId != sceneId ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeReplayFlow) ||
		user.MainQuest.ReplayFlowCurrentQuestSceneId != sceneId {
		t.Fatalf("replay-flow scene was not recorded: %+v", user.MainQuest)
	}
}

func mainQuestSceneProgressFixture(questId, sceneId int32) *QuestHandler {
	return &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			questId: {QuestId: questId, IsCountedAsQuest: true},
		},
		SceneById: map[int32]masterdata.EntityMQuestScene{
			sceneId: {QuestSceneId: sceneId, QuestId: questId},
		},
		SceneIdsByQuestId: map[int32][]int32{
			questId: {sceneId},
		},
	}}
}
