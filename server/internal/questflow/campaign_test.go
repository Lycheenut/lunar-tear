package questflow

import (
	"path/filepath"
	"testing"
	"time"

	"lunar-tear/server/internal/campaign"
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
