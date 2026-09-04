package questflow

import (
	"path/filepath"
	"testing"
	"time"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestCurrentMasterDataQuestCampaignUsesUserStatus(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatalf("init master data: %v", err)
	}
	campaigns, err := campaign.Load()
	if err != nil {
		t.Fatalf("load campaigns: %v", err)
	}
	h := &QuestHandler{Campaigns: campaigns}
	now := time.Date(2026, time.August, 10, 13, 30, 0, 0, time.UTC).UnixMilli()
	day := int64(24 * time.Hour / time.Millisecond)
	clearedUnlockQuest := map[int32]store.UserQuestState{
		1: {QuestStateType: model.UserQuestStateTypeCleared},
	}
	beginner := &store.UserState{
		RegisterDatetime: now - day,
		Quests:           clearedUnlockQuest,
	}
	comeback := &store.UserState{
		RegisterDatetime: now - 100*day,
		Login: store.UserLoginState{
			LastComebackLoginDatetime: now - day,
		},
		Quests: clearedUnlockQuest,
	}
	ordinary := &store.UserState{
		RegisterDatetime: now - 100*day,
		Quests:           clearedUnlockQuest,
	}
	target := campaign.QuestTarget{QuestType: campaign.QuestTypeMainQuest}

	if got := h.staminaWithCampaign(beginner, 10, target, now); got != 5 {
		t.Fatalf("beginner stamina = %d, want 5", got)
	}
	if got := h.staminaWithCampaign(comeback, 10, target, now); got != 5 {
		t.Fatalf("comeback stamina = %d, want 5", got)
	}
	if got := h.staminaWithCampaign(ordinary, 10, target, now); got != 10 {
		t.Fatalf("ordinary stamina = %d, want 10", got)
	}
	if got := h.appendBonusDrops(beginner, nil, target, now); len(got) == 0 {
		t.Fatal("beginner bonus drops are empty")
	}
	if got := h.appendBonusDrops(comeback, nil, target, now); len(got) == 0 {
		t.Fatal("comeback bonus drops are empty")
	}
	if got := h.appendBonusDrops(ordinary, nil, target, now); len(got) != 0 {
		t.Fatalf("ordinary bonus drops = %v, want none", got)
	}
}

func TestEventQuestSkipUsesEventStaminaCampaignAfterRecovery(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatalf("init master data: %v", err)
	}
	campaigns, err := campaign.Load()
	if err != nil {
		t.Fatalf("load campaigns: %v", err)
	}
	const (
		questId   = 10
		chapterId = 20
		now       = int64(1_647_500_000_000)
	)
	h := &QuestHandler{
		QuestCatalog: &masterdata.QuestCatalog{
			QuestById:                 map[int32]masterdata.EntityMQuest{questId: {QuestId: questId, Stamina: 10, IsUsableSkipTicket: true}},
			MaxStaminaByLevel:         map[int32]int32{1: 100},
			EventQuestTypeByChapterId: map[int32]int32{chapterId: 4},
			MissionIdsByQuestId:       map[int32][]int32{},
			BattleDropsByQuestId:      map[int32][]masterdata.BattleDropInfo{},
		},
		Config:    &masterdata.GameConfig{ConsumableItemIdForQuestSkipTicket: 7},
		Granter:   &store.PossessionGranter{},
		Campaigns: campaigns,
	}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Status.Level = 1
	user.Status.StaminaMilliValue = 0
	user.Status.StaminaUpdateDatetime = now
	user.ConsumableItems[7] = 1
	user.Quests[questId] = store.UserQuestState{QuestId: questId, QuestStateType: model.UserQuestStateTypeCleared}
	store.RecoverStamina(user, 5_000, 100_000, now)

	if _, err := h.HandleQuestSkip(user, questId, int32(model.QuestTypeEvent), chapterId, 0, 1, now); err != nil {
		t.Fatalf("event quest skip after stamina recovery failed: %v", err)
	}
	if user.Status.StaminaMilliValue != 0 || user.ConsumableItems[7] != 0 {
		t.Fatalf("skip costs = stamina %d, tickets %d; want both 0", user.Status.StaminaMilliValue, user.ConsumableItems[7])
	}
}
