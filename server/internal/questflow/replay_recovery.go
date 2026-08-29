package questflow

import (
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

// RecoverCompletedReplayOnLogin finishes the portal transition when the
// client disconnected after FinishMainQuest but before reporting the return
// to Mama's Room. Leaving this state in ReplayFlow makes the client try to
// restore the final, non-playable scene during startup.
func (h *QuestHandler) RecoverCompletedReplayOnLogin(user *store.UserState, nowMillis int64) bool {
	main := &user.MainQuest
	if !model.IsReplayQuestFlowType(main.CurrentQuestFlowType) ||
		main.ProgressQuestFlowType != main.CurrentQuestFlowType ||
		main.ProgressQuestSceneId != 0 ||
		main.ProgressHeadQuestSceneId != 0 ||
		main.SavedContext.Active ||
		main.ReplayFlowCurrentQuestSceneId == 0 ||
		main.ReplayFlowCurrentQuestSceneId != main.ReplayFlowHeadQuestSceneId {
		return false
	}

	scene, ok := h.SceneById[main.ReplayFlowCurrentQuestSceneId]
	if !ok || main.ReplayFlowCurrentQuestSceneId != h.getLastMainFlowSceneId(scene.QuestId) {
		return false
	}
	completedReplay := false
	for _, replayQuestId := range h.ReplayQuestIdsByMainQuestId[scene.QuestId] {
		replayQuest, ok := user.Quests[replayQuestId]
		if ok && replayQuest.QuestStateType == model.UserQuestStateTypeCleared && replayQuest.ClearCount > 0 {
			completedReplay = true
			break
		}
	}
	if !completedReplay {
		return false
	}

	main.CurrentQuestFlowType = int32(model.QuestFlowTypeMainFlow)
	main.LatestVersion = nowMillis
	user.PortalCageStatus.IsCurrentProgress = true
	user.PortalCageStatus.LatestVersion = nowMillis
	return true
}
