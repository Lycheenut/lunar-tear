package questflow

import (
	"fmt"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func (h *QuestHandler) validateQuestStart(user *store.UserState, questId int32, nowMillis int64) error {
	quest, ok := h.QuestById[questId]
	if !ok {
		return fmt.Errorf("unknown quest %d", questId)
	}
	cost := h.staminaWithCampaign(quest.Stamina, h.targetForMain(questId), nowMillis)
	if !store.HasEnoughStamina(user, cost, h.MaxStaminaByLevel[user.Status.Level]*1000, nowMillis) {
		return fmt.Errorf("insufficient stamina")
	}
	return nil
}

func (h *QuestHandler) validateExtraQuestStart(user *store.UserState, questId int32, nowMillis int64) error {
	quest, ok := h.QuestById[questId]
	if !ok {
		return fmt.Errorf("unknown quest %d", questId)
	}
	cost := h.staminaWithCampaign(quest.Stamina, h.targetForExtra(questId), nowMillis)
	if !store.HasEnoughStamina(user, cost, h.MaxStaminaByLevel[user.Status.Level]*1000, nowMillis) {
		return fmt.Errorf("insufficient stamina")
	}
	return nil
}

func (h *QuestHandler) validateBigHuntQuestStart(user *store.UserState, questId int32, nowMillis int64) error {
	quest, ok := h.QuestById[questId]
	if !ok {
		return fmt.Errorf("unknown quest %d", questId)
	}
	cost := h.staminaWithCampaign(quest.Stamina, h.targetForBigHunt(questId), nowMillis)
	if !store.HasEnoughStamina(user, cost, h.MaxStaminaByLevel[user.Status.Level]*1000, nowMillis) {
		return fmt.Errorf("insufficient stamina")
	}
	return nil
}

func (h *QuestHandler) EventChapterAvailable(user *store.UserState, chapterId int32, nowMillis int64) error {
	chapter, ok := h.EventChapterById[chapterId]
	if !ok {
		return fmt.Errorf("unknown event quest chapter %d", chapterId)
	}
	if nowMillis < chapter.StartDatetime || (chapter.EndDatetime > 0 && nowMillis >= chapter.EndDatetime) {
		return fmt.Errorf("event quest chapter %d is outside its active period", chapterId)
	}
	for _, questId := range h.EventUnlockQuestIdsByType[chapter.EventQuestType] {
		quest := user.Quests[questId]
		if quest.QuestStateType != model.UserQuestStateTypeCleared {
			return fmt.Errorf("event quest chapter %d is locked", chapterId)
		}
	}
	return nil
}

func (h *QuestHandler) validateEventQuest(user *store.UserState, chapterId, questId int32, nowMillis int64) error {
	if err := h.EventChapterAvailable(user, chapterId, nowMillis); err != nil {
		return err
	}
	if h.EventChapterIdByQuestId[questId] != chapterId {
		return fmt.Errorf("quest %d does not belong to event chapter %d", questId, chapterId)
	}
	quest := h.QuestById[questId]
	cost := h.staminaWithCampaign(quest.Stamina, h.targetForEvent(chapterId, questId), nowMillis)
	if !store.HasEnoughStamina(user, cost, h.MaxStaminaByLevel[user.Status.Level]*1000, nowMillis) {
		return fmt.Errorf("insufficient stamina")
	}
	return nil
}

func (h *QuestHandler) ValidateEventQuestContinuation(user *store.UserState, chapterId, questId int32, nowMillis int64) error {
	if err := h.EventChapterAvailable(user, chapterId, nowMillis); err != nil {
		return err
	}
	if h.EventChapterIdByQuestId[questId] != chapterId {
		return fmt.Errorf("quest %d does not belong to event chapter %d", questId, chapterId)
	}
	if user.EventQuest.CurrentEventQuestChapterId != chapterId || user.EventQuest.CurrentQuestId != questId {
		return fmt.Errorf("event quest %d is not active", questId)
	}
	return nil
}

func (h *QuestHandler) validateQuestSkip(user *store.UserState, questId, skipCount int32, nowMillis int64) error {
	quest, ok := h.QuestById[questId]
	if !ok {
		return fmt.Errorf("unknown quest %d", questId)
	}
	if skipCount <= 0 {
		return fmt.Errorf("skip count must be positive")
	}
	state := user.Quests[questId]
	if state.QuestStateType != model.UserQuestStateTypeCleared {
		return fmt.Errorf("quest %d is not cleared", questId)
	}
	if !quest.IsUsableSkipTicket {
		return fmt.Errorf("quest %d cannot be skipped", questId)
	}
	if quest.DailyClearableCount > 0 && state.DailyClearCount+skipCount > quest.DailyClearableCount {
		return fmt.Errorf("daily clear limit exceeded")
	}
	if user.ConsumableItems[h.Config.ConsumableItemIdForQuestSkipTicket] < skipCount {
		return fmt.Errorf("insufficient skip tickets")
	}
	cost, err := checkedProduct(h.staminaWithCampaign(quest.Stamina, h.targetForMain(questId), nowMillis), skipCount)
	if err != nil {
		return err
	}
	if !store.HasEnoughStamina(user, cost, h.MaxStaminaByLevel[user.Status.Level]*1000, nowMillis) {
		return fmt.Errorf("insufficient stamina")
	}
	return nil
}

func (h *QuestHandler) validateQuestSkipBulk(user *store.UserState, questIds, skipCounts []int32, nowMillis int64) error {
	if len(questIds) == 0 || len(questIds) != len(skipCounts) {
		return fmt.Errorf("invalid bulk skip request")
	}
	countsByQuest := make(map[int32]int64, len(questIds))
	for i, questId := range questIds {
		count := skipCounts[i]
		if count <= 0 {
			return fmt.Errorf("skip count must be positive")
		}
		countsByQuest[questId] += int64(count)
		if countsByQuest[questId] > int64(^uint32(0)>>1) {
			return fmt.Errorf("skip count is too large")
		}
	}
	var totalTickets, totalStamina int64
	for questId, totalCount := range countsByQuest {
		quest, ok := h.QuestById[questId]
		if !ok {
			return fmt.Errorf("unknown quest %d", questId)
		}
		count := int32(totalCount)
		state := user.Quests[questId]
		if state.QuestStateType != model.UserQuestStateTypeCleared || !quest.IsUsableSkipTicket {
			return fmt.Errorf("quest %d cannot be skipped", questId)
		}
		if quest.DailyClearableCount > 0 && state.DailyClearCount+count > quest.DailyClearableCount {
			return fmt.Errorf("daily clear limit exceeded")
		}
		totalTickets += totalCount
		totalStamina += int64(h.staminaWithCampaign(quest.Stamina, h.targetForMain(questId), nowMillis)) * totalCount
		if totalTickets > int64(^uint32(0)>>1) || totalStamina > int64(^uint32(0)>>1) {
			return fmt.Errorf("bulk skip cost is too large")
		}
	}
	if int64(user.ConsumableItems[h.Config.ConsumableItemIdForQuestSkipTicket]) < totalTickets {
		return fmt.Errorf("insufficient skip tickets")
	}
	if !store.HasEnoughStamina(user, int32(totalStamina), h.MaxStaminaByLevel[user.Status.Level]*1000, nowMillis) {
		return fmt.Errorf("insufficient stamina")
	}
	return nil
}

func checkedProduct(left, right int32) (int32, error) {
	value := int64(left) * int64(right)
	if left < 0 || right < 0 || value > int64(^uint32(0)>>1) {
		return 0, fmt.Errorf("cost is too large")
	}
	return int32(value), nil
}
