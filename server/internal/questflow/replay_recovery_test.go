package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestRecoverCompletedReplayOnLoginReturnsToPortal(t *testing.T) {
	h, user := completedReplayRecoveryFixture()

	if !h.RecoverCompletedReplayOnLogin(user, 456) {
		t.Fatal("completed replay was not recovered")
	}
	if user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeMainFlow) ||
		!user.PortalCageStatus.IsCurrentProgress ||
		user.MainQuest.LatestVersion != 456 ||
		user.PortalCageStatus.LatestVersion != 456 {
		t.Fatalf("completed replay did not return to portal: %+v", user.MainQuest)
	}
	if user.MainQuest.ReplayFlowCurrentQuestSceneId != 848 ||
		user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeReplayFlow) {
		t.Fatalf("sticky replay fields were changed: %+v", user.MainQuest)
	}
}

func TestRecoverCompletedReplayOnLoginPreservesIncompleteReplay(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*store.UserState)
	}{
		{
			name: "battle still active",
			mutate: func(user *store.UserState) {
				user.MainQuest.ProgressQuestSceneId = 845
				user.MainQuest.ProgressHeadQuestSceneId = 845
			},
		},
		{
			name: "ending scene not reached",
			mutate: func(user *store.UserState) {
				user.MainQuest.ReplayFlowCurrentQuestSceneId = 845
				user.MainQuest.ReplayFlowHeadQuestSceneId = 845
			},
		},
		{
			name: "replay quest not cleared",
			mutate: func(user *store.UserState) {
				replay := user.Quests[30330]
				replay.QuestStateType = model.UserQuestStateTypeActive
				user.Quests[30330] = replay
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, user := completedReplayRecoveryFixture()
			tt.mutate(user)

			if h.RecoverCompletedReplayOnLogin(user, 456) {
				t.Fatal("incomplete replay was recovered")
			}
			if user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeReplayFlow) ||
				user.PortalCageStatus.IsCurrentProgress {
				t.Fatalf("incomplete replay was changed: %+v", user.MainQuest)
			}
		})
	}
}

func completedReplayRecoveryFixture() (*QuestHandler, *store.UserState) {
	const (
		mainQuestId   = int32(334)
		replayQuestId = int32(30330)
	)
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		SceneById: map[int32]masterdata.EntityMQuestScene{
			845: {QuestSceneId: 845, QuestId: mainQuestId, SortOrder: 10, IsMainFlowQuestTarget: true},
			848: {QuestSceneId: 848, QuestId: mainQuestId, SortOrder: 20},
		},
		SceneIdsByQuestId:           map[int32][]int32{mainQuestId: {845, 848}},
		ReplayQuestIdsByMainQuestId: map[int32][]int32{mainQuestId: {replayQuestId}},
	}}
	user := store.SeedUserState(2, "test", 1, model.ClientPlatform{})
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ProgressQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ReplayFlowCurrentQuestSceneId = 848
	user.MainQuest.ReplayFlowHeadQuestSceneId = 848
	user.PortalCageStatus.IsCurrentProgress = false
	user.Quests[replayQuestId] = store.UserQuestState{
		QuestId:        replayQuestId,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     2,
	}
	return h, user
}
