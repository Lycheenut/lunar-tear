package questflow

import (
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

// RemoveReplayQuestMissionResidue removes rows created by older servers for
// replay-variant quest ids. The client has no corresponding mission master
// rows for those variants and cannot load the user table when they remain.
func (h *QuestHandler) RemoveReplayQuestMissionResidue(user *store.UserState) int {
	removed := 0
	for key := range user.QuestMissions {
		if h.isReplayQuestId(key.QuestId) {
			delete(user.QuestMissions, key)
			removed++
		}
	}
	return removed
}

// RecoverCompletedReplayOnLogin finishes the portal transition when the
// client disconnected after FinishMainQuest but before reporting the return
// to Mama's Room. Leaving this state in ReplayFlow makes the client try to
// restore the final, non-playable scene during startup.
func (h *QuestHandler) RecoverCompletedReplayOnLogin(user *store.UserState, nowMillis int64) bool {
	main := &user.MainQuest
	replayFlowType := main.ProgressQuestFlowType
	activeReplay := main.CurrentQuestFlowType == replayFlowType
	partialPortalTransition := main.CurrentQuestFlowType == int32(model.QuestFlowTypeMainFlow) &&
		user.PortalCageStatus.IsCurrentProgress
	if !model.IsReplayQuestFlowType(replayFlowType) ||
		(!activeReplay && !partialPortalTransition) ||
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
	main.ProgressQuestFlowType = int32(model.QuestFlowTypeUnknown)
	main.ReplayFlowCurrentQuestSceneId = 0
	main.ReplayFlowHeadQuestSceneId = 0
	main.LatestVersion = nowMillis
	user.PortalCageStatus.IsCurrentProgress = true
	user.PortalCageStatus.LatestVersion = nowMillis
	return true
}
