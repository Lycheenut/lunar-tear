package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestReplayQuestFinishAdvancesReplayFlowPastBattleResult(t *testing.T) {
	const (
		mainFlowQuestId = int32(334)
		replayQuestId   = int32(30330)
		battleResultId  = int32(845)
		finalSceneId    = int32(848)
	)
	h := &QuestHandler{
		QuestCatalog: &masterdata.QuestCatalog{
			QuestById: map[int32]masterdata.EntityMQuest{
				replayQuestId: {QuestId: replayQuestId, RecommendedDeckPower: 95000, IsCountedAsQuest: true},
			},
			MainFlowQuestIdByQuestId: map[int32]int32{replayQuestId: mainFlowQuestId},
			SceneIdsByQuestId:        map[int32][]int32{mainFlowQuestId: {battleResultId, finalSceneId}},
			SceneById: map[int32]masterdata.EntityMQuestScene{
				battleResultId: {QuestSceneId: battleResultId, QuestId: mainFlowQuestId, SortOrder: 10},
				finalSceneId:   {QuestSceneId: finalSceneId, QuestId: mainFlowQuestId, SortOrder: 20},
			},
			UserExpThresholds: []int32{0},
		},
		Config:  &masterdata.GameConfig{},
		Granter: &store.PossessionGranter{},
	}
	user := store.SeedUserState(2, "test", 1, model.ClientPlatform{})
	user.Quests[replayQuestId] = store.UserQuestState{
		QuestId:        replayQuestId,
		QuestStateType: model.UserQuestStateTypeActive,
	}
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ProgressQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ProgressQuestSceneId = battleResultId
	user.MainQuest.ProgressHeadQuestSceneId = battleResultId
	user.MainQuest.ReplayFlowCurrentQuestSceneId = battleResultId
	user.MainQuest.ReplayFlowHeadQuestSceneId = battleResultId

	h.HandleQuestFinish(user, replayQuestId, false, false, 123)

	if got := user.Quests[replayQuestId].QuestStateType; got != model.UserQuestStateTypeCleared {
		t.Fatalf("replay quest state = %d, want cleared", got)
	}
	if user.MainQuest.ProgressQuestSceneId != 0 || user.MainQuest.ProgressHeadQuestSceneId != 0 {
		t.Fatalf("active progress was not cleared: %+v", user.MainQuest)
	}
	if user.MainQuest.ReplayFlowCurrentQuestSceneId != finalSceneId ||
		user.MainQuest.ReplayFlowHeadQuestSceneId != finalSceneId {
		t.Fatalf("replay flow remained on battle result: %+v", user.MainQuest)
	}
	if user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeReplayFlow) ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeReplayFlow) {
		t.Fatalf("replay flow type changed before result UI completed: %+v", user.MainQuest)
	}
}

func TestRetiredReplayQuestDoesNotAdvanceReplayFlow(t *testing.T) {
	const (
		mainFlowQuestId = int32(334)
		replayQuestId   = int32(30330)
		battleResultId  = int32(845)
		finalSceneId    = int32(848)
	)
	h := &QuestHandler{
		QuestCatalog: &masterdata.QuestCatalog{
			QuestById: map[int32]masterdata.EntityMQuest{
				replayQuestId: {QuestId: replayQuestId, RecommendedDeckPower: 95000, IsCountedAsQuest: true},
			},
			MainFlowQuestIdByQuestId: map[int32]int32{replayQuestId: mainFlowQuestId},
			SceneIdsByQuestId:        map[int32][]int32{mainFlowQuestId: {battleResultId, finalSceneId}},
			SceneById: map[int32]masterdata.EntityMQuestScene{
				battleResultId: {QuestSceneId: battleResultId, QuestId: mainFlowQuestId, SortOrder: 10},
				finalSceneId:   {QuestSceneId: finalSceneId, QuestId: mainFlowQuestId, SortOrder: 20},
			},
		},
	}
	user := store.SeedUserState(2, "test", 1, model.ClientPlatform{})
	user.Quests[replayQuestId] = store.UserQuestState{
		QuestId:        replayQuestId,
		QuestStateType: model.UserQuestStateTypeActive,
	}
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ReplayFlowCurrentQuestSceneId = battleResultId
	user.MainQuest.ReplayFlowHeadQuestSceneId = battleResultId

	h.HandleQuestFinish(user, replayQuestId, true, false, 123)

	if user.MainQuest.ReplayFlowCurrentQuestSceneId != battleResultId ||
		user.MainQuest.ReplayFlowHeadQuestSceneId != battleResultId {
		t.Fatalf("retired replay flow advanced: %+v", user.MainQuest)
	}
}
