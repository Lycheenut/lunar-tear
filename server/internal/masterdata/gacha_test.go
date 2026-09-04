package masterdata

import (
	"fmt"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestChapterGachaCatalogContainsMetadataWithoutRewards(t *testing.T) {
	entries := buildChapterGachaEntries()
	wantChapterIds := [...]int32{
		2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 13, 14,
		17, 18, 19, 20, 21, 22, 25, 26, 27, 28, 29, 30,
	}
	if len(entries) != len(wantChapterIds)+1 {
		t.Fatalf("chapter Gacha count = %d, want %d", len(entries), len(wantChapterIds)+1)
	}

	for i, entry := range entries[:len(wantChapterIds)] {
		chapterNumber := int32(i + 1)
		if entry.GachaId != chapterGachaIdBase+chapterNumber ||
			entry.BannerAssetName != fmt.Sprintf("chapter_%d", chapterNumber) ||
			entry.RelatedMainQuestChapterId != wantChapterIds[i] ||
			entry.GachaLabelType != model.GachaLabelChapter ||
			entry.GachaModeType != model.GachaModeBox ||
			entry.GachaAutoResetType != model.GachaAutoResetMonthly {
			t.Fatalf("unexpected chapter %d catalog entry: %+v", chapterNumber, entry)
		}
		if len(entry.PricePhases) != 2 ||
			entry.PricePhases[0].PriceId != 1008+int32(i) ||
			entry.PricePhases[1].PriceId != 1008+int32(i) ||
			entry.PricePhases[0].RegularPrice != 0 ||
			entry.PricePhases[1].RegularPrice != 0 {
			t.Fatalf("chapter %d price phases use the wrong ticket: %+v", chapterNumber, entry.PricePhases)
		}
		if len(entry.BoxItems) != 0 || len(entry.PromotionItems) != 0 || entry.BoxCount != 0 {
			t.Fatalf("chapter %d contains hardcoded rewards: %+v", chapterNumber, entry)
		}
	}

	ex := entries[len(entries)-1]
	if ex.GachaId != chapterGachaIdBase || ex.BannerAssetName != "chapter_ex" ||
		ex.RelatedMainQuestChapterId != 0 || ex.RelatedEventQuestChapterId != trueDarkMemoryQuestChapterId ||
		len(ex.PricePhases) != 2 ||
		ex.PricePhases[0].PriceId != 2002 || ex.PricePhases[1].PriceId != 2002 {
		t.Fatalf("unexpected true-dark chapter catalog entry: %+v", ex)
	}
	if len(ex.BoxItems) != 0 || len(ex.PromotionItems) != 0 || ex.BoxCount != 0 {
		t.Fatalf("true-dark chapter contains hardcoded rewards: %+v", ex)
	}
}

func TestChapterGachaUnlocksWhenPlayerReachesChapter(t *testing.T) {
	entries := buildChapterGachaEntries()[:3]
	quests := &QuestCatalog{
		OrderedQuestIds:             []int32{20, 21, 30, 31},
		MainQuestChapterIdByQuestId: map[int32]int32{20: 2, 21: 2, 30: 3, 31: 3},
	}
	EnrichGachaUnlockConditions(entries, quests)

	if got := entries[0].UnlockConditions; len(got) != 1 || got[0].GachaUnlockConditionType != model.GachaUnlockNone {
		t.Fatalf("chapter 1 unlock conditions = %+v, want unlocked", got)
	}
	if got := entries[1].UnlockConditions; len(got) != 1 || got[0].ConditionValue != 21 {
		t.Fatalf("chapter 2 unlock conditions = %+v, want final quest 21 of chapter 1", got)
	}
	if got := entries[2].UnlockConditions; len(got) != 1 || got[0].ConditionValue != 31 {
		t.Fatalf("chapter 3 unlock conditions = %+v, want final quest 31 of chapter 2", got)
	}
}

func TestTrueDarkChapterGachaIsUnconditionallyUnlocked(t *testing.T) {
	entries := buildChapterGachaEntries()
	trueDarkIndex := len(entries) - 1
	quests := &QuestCatalog{
		OrderedQuestIds:             []int32{130},
		MainQuestChapterIdByQuestId: map[int32]int32{130: 13},
	}
	EnrichGachaUnlockConditions(entries, quests)

	conditions := entries[trueDarkIndex].UnlockConditions
	if len(conditions) != 1 || conditions[0].GachaUnlockConditionType != model.GachaUnlockNone {
		t.Fatalf("true-dark chapter unlock conditions = %+v, want unlocked", conditions)
	}
}

func TestChapterGachaBranchesUseTheirOwnPreviousChapter(t *testing.T) {
	entries := []store.GachaCatalogEntry{
		{GachaLabelType: model.GachaLabelChapter, RelatedMainQuestChapterId: 14},
		{GachaLabelType: model.GachaLabelChapter, RelatedMainQuestChapterId: 17},
		{GachaLabelType: model.GachaLabelChapter, RelatedMainQuestChapterId: 18},
		{GachaLabelType: model.GachaLabelChapter, RelatedMainQuestChapterId: 25},
		{GachaLabelType: model.GachaLabelChapter, RelatedMainQuestChapterId: 26},
	}
	quests := &QuestCatalog{
		OrderedQuestIds: []int32{140, 170, 180, 250, 260},
		MainQuestChapterIdByQuestId: map[int32]int32{
			140: 14, 170: 17, 180: 18, 250: 25, 260: 26,
		},
		MainQuestRouteIdByChapterId: map[int32]int32{
			14: 1, 17: 2, 18: 2, 25: 3, 26: 3,
		},
		SeasonIdByRouteId: map[int32]int32{1: 1, 2: 2, 3: 2},
	}
	EnrichGachaUnlockConditions(entries, quests)

	wantPrerequisiteQuestIds := []int32{0, 140, 170, 140, 250}
	for i, want := range wantPrerequisiteQuestIds {
		conditions := entries[i].UnlockConditions
		if len(conditions) != 1 {
			t.Fatalf("chapter entry %d conditions = %+v, want one condition", i, conditions)
		}
		if want == 0 {
			if conditions[0].GachaUnlockConditionType != model.GachaUnlockNone {
				t.Fatalf("chapter entry %d condition = %+v, want unlocked", i, conditions[0])
			}
			continue
		}
		if conditions[0].GachaUnlockConditionType != model.GachaUnlockMainQuestClear || conditions[0].ConditionValue != want {
			t.Fatalf("chapter entry %d condition = %+v, want quest %d", i, conditions[0], want)
		}
	}
}

func TestLoadGachaCatalogIncludesEventMetadataWithoutSynthesizedInventory(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog(61)
	if err != nil {
		t.Fatal(err)
	}
	hasMamaBanner := false
	eventCount := 0
	for _, entry := range entries {
		if entry.GachaLabelType == model.GachaLabelPremium && entry.IsMamaBanner {
			t.Fatalf("premium Gacha %d was loaded from m_mom_banner", entry.GachaId)
		}
		if entry.GachaLabelType == model.GachaLabelEvent {
			eventCount++
			if len(entry.BoxItems) != 0 || entry.BoxCount != 0 || entry.RequiredConsumableItemId == 0 || entry.RelatedEventQuestChapterId == 0 {
				t.Fatalf("event gacha metadata is incomplete or synthesized inventory: %+v", entry)
			}
		}
		hasMamaBanner = hasMamaBanner || entry.IsMamaBanner
	}
	if len(entries) == 0 {
		t.Fatal("non-event gachas were removed with event gachas")
	}
	if !hasMamaBanner {
		t.Fatal("m_mom_banner entries were not marked as Mama banners")
	}
	if eventCount == 0 {
		t.Fatal("event Gacha metadata was not loaded")
	}
}

func TestLoadGachaCatalogIncludesGuaranteedTicketGachas(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog(61)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		gachaId       int32
		ticketId      int32
		assetName     string
		minimumRarity model.RarityType
	}{
		{model.GachaIdGuaranteedThreeStarOrHigher, model.ConsumableIdGuaranteedThreeStarOrHigherTicket, guaranteedThreeStarOrHigherGachaAssetName, model.RaritySRare},
		{model.GachaIdGuaranteedFourStar, model.ConsumableIdGuaranteedFourStarTicket, guaranteedFourStarGachaAssetName, model.RaritySSRare},
	}
	byId := make(map[int32]store.GachaCatalogEntry, len(entries))
	maxMasterSortOrder := int32(0)
	var catalogStartDatetime int64
	var catalogEndDatetime int64
	for _, entry := range entries {
		byId[entry.GachaId] = entry
		if model.IsGuaranteedTicketGacha(entry.GachaId) {
			continue
		}
		if entry.SortOrder > maxMasterSortOrder {
			maxMasterSortOrder = entry.SortOrder
		}
		if entry.StartDatetime > 0 && (catalogStartDatetime == 0 || entry.StartDatetime < catalogStartDatetime) {
			catalogStartDatetime = entry.StartDatetime
		}
		if entry.EndDatetime > catalogEndDatetime {
			catalogEndDatetime = entry.EndDatetime
		}
	}
	for _, tt := range tests {
		entry, ok := byId[tt.gachaId]
		if !ok {
			t.Fatalf("Gacha %d was not added to the catalog", tt.gachaId)
		}
		if entry.BannerAssetName != tt.assetName || entry.RequiredConsumableItemId != tt.ticketId {
			t.Fatalf("unexpected guaranteed Gacha %d: %+v", tt.gachaId, entry)
		}
		if entry.IsMamaBanner {
			t.Fatalf("ticket-only Gacha %d was marked as a Mama banner", tt.gachaId)
		}
		if entry.SortOrder <= maxMasterSortOrder {
			t.Fatalf("ticket-only Gacha %d sort order %d is not after master-data Gachas ending at %d", tt.gachaId, entry.SortOrder, maxMasterSortOrder)
		}
		if entry.StartDatetime != catalogStartDatetime || entry.EndDatetime != catalogEndDatetime {
			t.Fatalf("ticket-only Gacha %d availability = %d..%d, want catalog availability %d..%d", tt.gachaId, entry.StartDatetime, entry.EndDatetime, catalogStartDatetime, catalogEndDatetime)
		}
		if len(entry.PricePhases) != 1 {
			t.Fatalf("Gacha %d price phase count = %d, want 1", tt.gachaId, len(entry.PricePhases))
		}
		phase := entry.PricePhases[0]
		if phase.PhaseId != tt.gachaId*model.PhaseIdMultiplier+1 ||
			phase.PriceType != model.PriceTypeConsumableItem ||
			phase.PriceId != tt.ticketId ||
			phase.Price != 1 || phase.RegularPrice != 0 || phase.DrawCount != 1 ||
			phase.FixedRarityMin != tt.minimumRarity || phase.FixedCount != 1 {
			t.Fatalf("unexpected guaranteed Gacha %d price phase: %+v", tt.gachaId, phase)
		}
	}
	if byId[model.GachaIdGuaranteedFourStar].SortOrder <= byId[model.GachaIdGuaranteedThreeStarOrHigher].SortOrder {
		t.Fatal("four-star guaranteed Gacha was not ordered after the three-star guaranteed Gacha")
	}
}

func TestLoadGachaCatalogIncludesDailyGachaWithConfiguredUnlockQuest(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog(61)
	if err != nil {
		t.Fatal(err)
	}
	var daily *store.GachaCatalogEntry
	for i := range entries {
		if entries[i].GachaId == model.GachaIdDaily {
			daily = &entries[i]
			break
		}
	}
	if daily == nil {
		t.Fatal("daily Gacha was not added to the catalog")
	}
	if daily.BannerAssetName != dailyGachaAssetName || daily.IsMamaBanner ||
		daily.GachaLabelType != model.GachaLabelPremium || daily.GachaModeType != model.GachaModeBasic ||
		daily.GachaAutoResetType != model.GachaAutoResetDaily || daily.GachaAutoResetPeriod != 1 {
		t.Fatalf("unexpected daily Gacha: %+v", *daily)
	}
	if len(daily.UnlockConditions) != 1 ||
		daily.UnlockConditions[0].GachaUnlockConditionType != model.GachaUnlockMainQuestClear ||
		daily.UnlockConditions[0].ConditionValue != 61 {
		t.Fatalf("daily Gacha unlock conditions = %+v, want quest 61", daily.UnlockConditions)
	}
	if len(daily.PricePhases) != 1 {
		t.Fatalf("daily Gacha price phase count = %d, want 1", len(daily.PricePhases))
	}
	phase := daily.PricePhases[0]
	if phase.Price != 0 || phase.DrawCount != model.DailyGachaDrawCount || phase.LimitExecCount != model.DailyGachaExecLimit {
		t.Fatalf("unexpected daily Gacha price phase: %+v", phase)
	}
}
