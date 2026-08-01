package questflow

import (
	"fmt"
	"log"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func (h *QuestHandler) HandleBigHuntQuestStart(user *store.UserState, questId, userDeckNumber int32, nowMillis int64) error {
	quest, ok := h.QuestById[questId]
	if !ok {
		return fmt.Errorf("unknown quest %d", questId)
	}
	if err := h.validateBigHuntQuestStart(user, questId, nowMillis); err != nil {
		return err
	}

	h.initQuestState(user, questId)

	if quest.Stamina > 0 {
		maxMillis := h.MaxStaminaByLevel[user.Status.Level] * 1000
		stamina := h.staminaWithCampaign(quest.Stamina, h.targetForBigHunt(questId), nowMillis)
		if err := store.ConsumeStamina(user, stamina, maxMillis, nowMillis); err != nil {
			return err
		}
	}

	questState := user.Quests[questId]
	questState.UserDeckNumber = userDeckNumber
	questState.QuestStateType = model.UserQuestStateTypeActive
	questState.LatestStartDatetime = nowMillis
	user.Quests[questId] = questState
	return nil
}

func (h *QuestHandler) HandleBigHuntQuestFinish(user *store.UserState, questId int32, isRetired, isAnnihilated bool, nowMillis int64) FinishOutcome {
	quest, ok := h.QuestById[questId]
	if !ok {
		log.Printf("[HandleBigHuntQuestFinish] unknown questId=%d", questId)
		return FinishOutcome{}
	}

	target := h.targetForBigHunt(questId)
	var outcome FinishOutcome
	if !isRetired && !isAnnihilated {
		outcome = h.evaluateFinishOutcome(user, questId, target, nowMillis)
		h.applyQuestVictory(user, questId, &outcome, nowMillis, false)
	}

	consumed := h.staminaWithCampaign(quest.Stamina, target, nowMillis)
	if isRetired && !isAnnihilated && consumed > 1 {
		refund := consumed - 1
		maxMillis := h.MaxStaminaByLevel[user.Status.Level] * 1000
		store.RecoverStamina(user, refund*1000, maxMillis, nowMillis)
	}

	return outcome
}
