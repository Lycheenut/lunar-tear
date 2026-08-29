package questflow

// MainQuest scene-field families mirror three client entity tables:
//
//	MainFlow*    — EntityIUserMainQuestMainFlowStatus (#11443)
//	Progress*    — EntityIUserMainQuestProgressStatus (#11444)
//	ReplayFlow*  — EntityIUserMainQuestReplayFlowStatus (#11445)

import (
	"fmt"
	"log"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func (h *QuestHandler) applySceneGrants(user *store.UserState, questSceneId int32, nowMillis int64) {
	grants, ok := h.SceneGrantsBySceneId[questSceneId]
	if !ok {
		return
	}
	for _, g := range grants {
		h.applyRewardPossession(user, model.PossessionType(g.PossessionType), g.PossessionId, g.Count, nowMillis)
	}
}

func (h *QuestHandler) isSceneAhead(newSceneId, currentHeadId int32) bool {
	if currentHeadId == 0 {
		return true
	}
	return h.SceneById[newSceneId].SortOrder > h.SceneById[currentHeadId].SortOrder
}

func (h *QuestHandler) advanceMainFlowScene(user *store.UserState, questId, sceneId int32) {
	if !h.isSceneAhead(sceneId, user.MainQuest.HeadQuestSceneId) {
		return
	}
	user.MainQuest.CurrentQuestSceneId = sceneId
	user.MainQuest.HeadQuestSceneId = sceneId

	lastSceneId := h.getChapterLastSceneId(questId)
	user.MainQuest.IsReachedLastQuestScene = sceneId == lastSceneId

	if routeId, ok := h.RouteIdByQuestId[questId]; ok {
		user.MainQuest.CurrentMainQuestRouteId = routeId
		if seasonId, ok := h.SeasonIdByRouteId[routeId]; ok {
			user.MainQuest.MainQuestSeasonId = seasonId
		}
	}
}

func (h *QuestHandler) advanceReplayFlowScene(user *store.UserState, sceneId int32) {
	if !h.isSceneAhead(sceneId, user.MainQuest.ReplayFlowHeadQuestSceneId) {
		return
	}
	user.MainQuest.ReplayFlowCurrentQuestSceneId = sceneId
	user.MainQuest.ReplayFlowHeadQuestSceneId = sceneId
}

func (h *QuestHandler) SeasonRoutesFor(user *store.UserState) map[int32]int32 {
	out := make(map[int32]int32)
	for seasonId, routes := range h.RoutesBySeason {
		if seasonId <= 1 {
			continue
		}
		for _, routeId := range routes {
			finalQuestId, ok := h.RouteCompletionQuestId[routeId]
			if !ok {
				continue
			}
			if q, ok := user.Quests[finalQuestId]; ok && q.ClearCount > 0 {
				out[seasonId] = routeId
				break
			}
		}
	}
	if cur := user.MainQuest.MainQuestSeasonId; cur >= 2 && user.MainQuest.CurrentMainQuestRouteId > 0 {
		out[cur] = user.MainQuest.CurrentMainQuestRouteId
	}
	return out
}

func (h *QuestHandler) HandleMainFlowSceneProgress(user *store.UserState, questSceneId int32, nowMillis int64) error {
	scene, ok := h.SceneById[questSceneId]
	if !ok {
		return fmt.Errorf("unknown scene %d", questSceneId)
	}

	quest, ok := h.QuestById[scene.QuestId]
	if !ok {
		return fmt.Errorf("unknown quest %d for scene %d", scene.QuestId, questSceneId)
	}

	h.advanceMainFlowScene(user, quest.QuestId, questSceneId)
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeMainFlow)

	if user.SideStoryActiveProgress.CurrentSideStoryQuestId != 0 {
		user.SideStoryActiveProgress = store.SideStoryActiveProgress{
			LatestVersion: nowMillis,
		}
	}

	user.PortalCageStatus.IsCurrentProgress = false
	user.PortalCageStatus.LatestVersion = nowMillis

	h.applySceneGrants(user, questSceneId, nowMillis)
	return nil
}

func (h *QuestHandler) advanceTutorialsForScene(user *store.UserState, sceneId int32) {
	currentScene, ok := h.SceneById[sceneId]
	if !ok {
		log.Printf("[advanceTutorialsForScene] unknown sceneId=%d", sceneId)
		return
	}
	for _, cond := range h.TutorialUnlockConditions {
		condScene, ok := h.SceneById[cond.ConditionValue]
		if !ok {
			log.Printf("[advanceTutorialsForScene] unknown conditionValue=%d", cond.ConditionValue)
			continue
		}
		if currentScene.SortOrder >= condScene.SortOrder {
			if _, exists := user.Tutorials[cond.TutorialType]; !exists {
				user.Tutorials[cond.TutorialType] = store.TutorialProgressState{
					TutorialType:  cond.TutorialType,
					ProgressPhase: 99999,
				}
			}
		}
	}
}

func (h *QuestHandler) getLastMainFlowSceneId(questId int32) int32 {
	sceneIds := h.SceneIdsByQuestId[questId]
	if len(sceneIds) == 0 {
		log.Printf("[getLastMainFlowSceneId] no scenes for questId=%d", questId)
		return 0
	}
	return sceneIds[len(sceneIds)-1]
}

func (h *QuestHandler) getChapterLastSceneId(questId int32) int32 {
	if id, ok := h.ChapterLastSceneByQuestId[questId]; ok {
		return id
	}
	return h.getLastMainFlowSceneId(questId)
}

func (h *QuestHandler) HandleReplayFlowSceneProgress(user *store.UserState, questSceneId int32, nowMillis int64) {
	user.MainQuest.ReplayFlowCurrentQuestSceneId = questSceneId
	user.MainQuest.ReplayFlowHeadQuestSceneId = questSceneId

	user.PortalCageStatus.IsCurrentProgress = false
	user.PortalCageStatus.LatestVersion = nowMillis

	if user.SideStoryActiveProgress.CurrentSideStoryQuestId != 0 {
		user.SideStoryActiveProgress = store.SideStoryActiveProgress{
			LatestVersion: nowMillis,
		}
	}

	flowType := h.replayFlowType(user, questSceneId)
	user.MainQuest.CurrentQuestFlowType = int32(flowType)
	user.MainQuest.LatestVersion = nowMillis
	log.Printf("[HandleReplayFlowSceneProgress] sceneId=%d flowType=%s", questSceneId, flowType)
}

func (h *QuestHandler) replayFlowType(user *store.UserState, questSceneId int32) model.QuestFlowType {
	scene, ok := h.SceneById[questSceneId]
	if !ok {
		return model.QuestFlowTypeReplayFlow
	}
	routeId, ok := h.RouteIdByQuestId[scene.QuestId]
	if !ok {
		return model.QuestFlowTypeReplayFlow
	}
	return h.replayFlowTypeForRoute(user, routeId)
}

func (h *QuestHandler) replayFlowTypeForRoute(user *store.UserState, routeId int32) model.QuestFlowType {
	seasonId, ok := h.SeasonIdByRouteId[routeId]
	if !ok {
		return model.QuestFlowTypeReplayFlow
	}
	pairs := h.SeasonRoutesFor(user)
	if recorded, ok := pairs[seasonId]; ok && recorded != routeId {
		return model.QuestFlowTypeAnotherRouteReplayFlow
	}
	return model.QuestFlowTypeReplayFlow
}

func (h *QuestHandler) replayFlowTypeFromQuestId(user *store.UserState, questId int32) model.QuestFlowType {
	routeId, ok := h.RouteIdByQuestId[questId]
	if !ok {
		return model.QuestFlowTypeReplayFlow
	}
	return h.replayFlowTypeForRoute(user, routeId)
}

func (h *QuestHandler) HandleMainQuestSceneProgress(user *store.UserState, questSceneId int32) error {
	scene, ok := h.SceneById[questSceneId]
	if !ok {
		return fmt.Errorf("unknown scene %d", questSceneId)
	}

	quest, ok := h.QuestById[scene.QuestId]
	if !ok {
		return fmt.Errorf("unknown quest %d for scene %d", scene.QuestId, questSceneId)
	}

	isReplay := model.IsReplayQuestFlowType(user.MainQuest.CurrentQuestFlowType)
	if isMainQuestPlayable(quest) && !isReplay {
		questState := user.Quests[quest.QuestId]
		isMenuReplay := false
		if user.MainQuest.SavedContext.Active {
			if progressScene, ok := h.SceneById[user.MainQuest.ProgressQuestSceneId]; ok {
				isMenuReplay = progressScene.QuestId == quest.QuestId
			}
		}
		if questState.QuestStateType != model.UserQuestStateTypeActive && !isMenuReplay {
			if questState.QuestStateType == model.UserQuestStateTypeCleared && questState.ClearCount > 0 {
				log.Printf("[HandleMainQuestSceneProgress] ignored stale cleared quest scene quest=%d scene=%d", quest.QuestId, questSceneId)
				return nil
			}
			return fmt.Errorf("quest %d is not active", quest.QuestId)
		}
	}

	if prevSceneId := user.MainQuest.ProgressQuestSceneId; prevSceneId != 0 {
		if prevScene, ok := h.SceneById[prevSceneId]; ok && prevScene.QuestId != quest.QuestId {
			// Skip if the previous quest is playable — it has its own FinishMainQuest;
			// chain-finalizing here would double-increment ClearCount.
			if prevQuest, ok := h.QuestById[prevScene.QuestId]; ok && !isMainQuestPlayable(prevQuest) {
				h.finalizeChainPreviousQuest(user, prevScene.QuestId, gametime.NowMillis())
			}
		}
	}

	if isMainQuestPlayable(quest) {
		user.MainQuest.ProgressQuestSceneId = questSceneId
		if h.isSceneAhead(questSceneId, user.MainQuest.ProgressHeadQuestSceneId) {
			user.MainQuest.ProgressHeadQuestSceneId = questSceneId
		}
		if isReplay {
			user.MainQuest.ProgressQuestFlowType = user.MainQuest.CurrentQuestFlowType
		} else {
			user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeSubFlow)
			user.MainQuest.ProgressQuestFlowType = int32(model.QuestFlowTypeSubFlow)
		}
	} else if !isReplay {
		// Background/non-playable quest: advance the MainFlow pointer — but not
		// during a replay, where the isReplay block below tracks the ReplayFlow
		// scene and the MainFlow pointer must stay on real main-story progress.
		user.MainQuest.CurrentQuestSceneId = questSceneId
		if h.isSceneAhead(questSceneId, user.MainQuest.HeadQuestSceneId) {
			user.MainQuest.HeadQuestSceneId = questSceneId
		}
		lastSceneId := h.getChapterLastSceneId(quest.QuestId)
		user.MainQuest.IsReachedLastQuestScene = questSceneId == lastSceneId
	}

	if isReplay {
		user.MainQuest.ReplayFlowCurrentQuestSceneId = questSceneId
		if h.isSceneAhead(questSceneId, user.MainQuest.ReplayFlowHeadQuestSceneId) {
			user.MainQuest.ReplayFlowHeadQuestSceneId = questSceneId
		}
		user.MainQuest.LatestVersion = gametime.NowMillis()
	}
	return nil
}
