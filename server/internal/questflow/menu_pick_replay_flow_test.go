package questflow

import (
	"fmt"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestMenuPickFromReplayResumesAndReturnsToOriginalScene(t *testing.T) {
	for _, previousFlow := range []model.QuestFlowType{model.QuestFlowTypeReplayFlow, model.QuestFlowTypeAnotherRouteReplayFlow} {
		for _, retired := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/retired=%v", previousFlow, retired), func(t *testing.T) {
				h := mainQuestSceneProgressFixture(71, 335)
				h.Config = &masterdata.GameConfig{}
				h.SceneById[336] = masterdata.EntityMQuestScene{QuestSceneId: 336, QuestId: 71, SortOrder: 2}
				h.SceneIdsByQuestId[71] = []int32{335, 336}
				user := store.SeedUserState(3, "menu-pick-replay", 1, model.ClientPlatform{})
				user.MainQuest.CurrentQuestFlowType = int32(previousFlow)
				user.MainQuest.ReplayFlowCurrentQuestSceneId = 42
				user.MainQuest.ReplayFlowHeadQuestSceneId = 42
				originalMain := user.MainQuest
				user.Quests[71] = store.UserQuestState{
					QuestId: 71, QuestStateType: model.UserQuestStateTypeCleared, ClearCount: 1,
				}

				if err := h.HandleQuestStart(user, 71, true, false, 2, 100); err != nil {
					t.Fatal(err)
				}
				if user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeMainFlow) ||
					user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeMainFlow) {
					t.Errorf("menu pick retained the previous replay flow: %+v", user.MainQuest)
				}
				if user.MainQuest.SavedContext.CurrentQuestFlowType != int32(previousFlow) {
					t.Fatal("previous flow was not saved before switching to the menu quest")
				}
				if err := h.HandleMainQuestSceneProgress(user, 335); err != nil {
					t.Fatal(err)
				}
				user.BattleBinary = []byte("menu quest checkpoint")
				if err := h.HandleQuestRestart(user, 71, 200); err != nil {
					t.Fatalf("resume menu quest: %v", err)
				}
				if string(user.BattleBinary) != "menu quest checkpoint" || user.Quests[71].UserDeckNumber != 2 {
					t.Fatal("resume lost the checkpoint or selected deck")
				}
				if err := h.HandleMainQuestSceneProgress(user, 336); err != nil {
					t.Fatal(err)
				}
				if user.MainQuest.CurrentQuestFlowType != int32(model.QuestFlowTypeSubFlow) ||
					user.MainQuest.ProgressQuestFlowType != int32(model.QuestFlowTypeSubFlow) ||
					user.MainQuest.ReplayFlowCurrentQuestSceneId != 42 || user.MainQuest.ReplayFlowHeadQuestSceneId != 42 {
					t.Errorf("menu quest progress overwrote the original replay: %+v", user.MainQuest)
				}
				if err := h.ValidateMainQuestContinuation(user, 71); err != nil {
					t.Fatalf("finish menu quest: %v", err)
				}
				h.HandleQuestFinish(user, 71, retired, false, 300)
				originalMain.LatestVersion = 300
				if user.MainQuest != originalMain {
					t.Errorf("finish did not restore the original scene: got %+v, want %+v", user.MainQuest, originalMain)
				}
				wantClears := int32(2)
				if retired {
					wantClears = 1
				}
				if got := user.Quests[71]; got.QuestStateType != model.UserQuestStateTypeCleared || got.ClearCount != wantClears {
					t.Errorf("incorrect menu quest completion: %+v", got)
				}
				if len(user.BattleBinary) != 0 {
					t.Fatal("finished menu quest retained a battle checkpoint")
				}
			})
		}
	}
}
