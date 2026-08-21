package importantitem

import (
	"path/filepath"
	"testing"
	"time"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
)

func loadTestCatalog(t *testing.T) *Catalog {
	t.Helper()
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatalf("initialize master data: %v", err)
	}
	catalog, err := Load()
	if err != nil {
		t.Fatalf("load important-item effects: %v", err)
	}
	return catalog
}

func TestLoadIncludesAllServerSideImportantItemEffects(t *testing.T) {
	catalog := loadTestCatalog(t)
	wantIds := map[int32]bool{
		200001: true, 200002: true, 200003: true, 200005: true, 200006: true,
		200007: true, 200008: true, 200009: true, 200010: true, 200011: true,
		200012: true, 200013: true, 200014: true, 200015: true, 200016: true,
		200017: true, 200018: true, 200019: true, 200020: true, 200021: true,
	}
	if got := catalog.EffectCount(); got != len(wantIds) {
		t.Fatalf("server-side important-item effect count = %d, want %d", got, len(wantIds))
	}
	for _, rule := range catalog.rules {
		if !wantIds[rule.importantItemId] {
			t.Fatalf("unexpected server-side important-item effect %d", rule.importantItemId)
		}
		delete(wantIds, rule.importantItemId)
	}
	if len(wantIds) != 0 {
		t.Fatalf("missing server-side important-item effects: %v", wantIds)
	}
}

func TestQuestBonusesMatchMasterDataTargets(t *testing.T) {
	catalog := loadTestCatalog(t)
	nowMillis := time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC).UnixMilli()
	tests := []struct {
		name          string
		importantItem int32
		quest         campaign.QuestTarget
		possession    model.PossessionType
		possessionId  int32
		wantRate      int32
		wantCount     int32
	}{
		{
			name:          "main chapter ticket count",
			importantItem: 200001,
			quest:         campaign.QuestTarget{QuestType: campaign.QuestTypeMainQuest, ChapterId: 2},
			possession:    model.PossessionTypeConsumableItem,
			possessionId:  1008,
			wantCount:     500,
		},
		{
			name:          "main chapter excludes other items",
			importantItem: 200001,
			quest:         campaign.QuestTarget{QuestType: campaign.QuestTypeMainQuest, ChapterId: 2},
			possession:    model.PossessionTypeConsumableItem,
			possessionId:  999,
		},
		{
			name:          "very hard main quest material count",
			importantItem: 200005,
			quest: campaign.QuestTarget{
				QuestType:               campaign.QuestTypeMainQuest,
				MainQuestDifficultyType: 3,
			},
			possession:   model.PossessionTypeMaterial,
			possessionId: 330001,
			wantCount:    500,
		},
		{
			name:          "specific event quest gem rate",
			importantItem: 200007,
			quest:         campaign.QuestTarget{QuestType: campaign.QuestTypeEventQuest, QuestId: 100009},
			possession:    model.PossessionTypeMaterial,
			possessionId:  321008,
			wantRate:      500,
		},
		{
			name:          "dungeon parts rate",
			importantItem: 200012,
			quest: campaign.QuestTarget{
				QuestType:      campaign.QuestTypeEventQuest,
				EventQuestType: 3,
			},
			possession:   model.PossessionTypeParts,
			possessionId: 3,
			wantRate:     500,
		},
		{
			name:          "big hunt daily reward count",
			importantItem: 200020,
			quest:         campaign.QuestTarget{QuestType: campaign.QuestTypeBigHunt},
			possession:    model.PossessionTypeMaterial,
			possessionId:  999,
			wantCount:     1000,
		},
		{
			name:          "dark memory all-item rate",
			importantItem: 200021,
			quest: campaign.QuestTarget{
				QuestType:      campaign.QuestTypeEventQuest,
				EventQuestType: 7,
			},
			possession:   model.PossessionTypeMaterial,
			possessionId: 999,
			wantRate:     1000,
		},
		{
			name:          "client-only auto effect is ignored",
			importantItem: 200004,
			quest:         campaign.QuestTarget{QuestType: campaign.QuestTypeMainQuest},
			possession:    model.PossessionTypeMaterial,
			possessionId:  999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, count := catalog.QuestBonuses(
				map[int32]int32{tt.importantItem: 1},
				tt.quest,
				tt.possession,
				tt.possessionId,
				nowMillis,
			)
			if rate != tt.wantRate || count != tt.wantCount {
				t.Fatalf("bonuses = (rate %d, count %d), want (rate %d, count %d)", rate, count, tt.wantRate, tt.wantCount)
			}
		})
	}
}

func TestQuestBonusesStackAndRespectEffectPeriod(t *testing.T) {
	catalog := loadTestCatalog(t)
	quest := campaign.QuestTarget{QuestType: campaign.QuestTypeEventQuest, QuestId: 100009}
	importantItems := map[int32]int32{200007: 1, 200014: 1}

	rate, count := catalog.QuestBonuses(
		importantItems,
		quest,
		model.PossessionTypeMaterial,
		321008,
		time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC).UnixMilli(),
	)
	if rate != 1300 || count != 0 {
		t.Fatalf("stacked bonuses = (rate %d, count %d), want (rate 1300, count 0)", rate, count)
	}

	rate, count = catalog.QuestBonuses(importantItems, quest, model.PossessionTypeMaterial, 321008, 1)
	if rate != 0 || count != 0 {
		t.Fatalf("bonuses before effect period = (rate %d, count %d), want zero", rate, count)
	}
}
