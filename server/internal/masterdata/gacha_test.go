package masterdata

import (
	"fmt"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestChapterGachaCatalogMatchesReconstructedChapters(t *testing.T) {
	entries := buildChapterGachaEntries()
	wantChapterIds := [...]int32{2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 13}
	wantWeightTotals := [...]int32{9990, 9990, 10000, 9990, 9990, 10000, 10004, 10004, 10000, 10000, 10000}
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
		wantRows := 22
		if chapterNumber == 7 || chapterNumber == 8 {
			wantRows = 23
		}
		if len(entry.BoxItems) != wantRows {
			t.Fatalf("chapter %d reward row count = %d, want %d", chapterNumber, len(entry.BoxItems), wantRows)
		}
		var weightTotal int32
		for row, item := range entry.BoxItems {
			weightTotal += item.Weight
			if item.CounterId != int32(row+1) {
				t.Fatalf("chapter %d row %d counter id = %d", chapterNumber, row, item.CounterId)
			}
		}
		if weightTotal != wantWeightTotals[i] {
			t.Fatalf("chapter %d weight total = %d, want %d", chapterNumber, weightTotal, wantWeightTotals[i])
		}
	}

	first := entries[0].BoxItems[0]
	if first.PossessionId != 100004 || first.Count != 5 || first.MaxCount != 20 || first.Weight != 200 {
		t.Fatalf("unexpected first chapter reward row: %+v", first)
	}

	trueDark := entries[len(entries)-1]
	if trueDark.GachaId != chapterGachaIdBase || trueDark.BannerAssetName != "chapter_ex" ||
		trueDark.RelatedMainQuestChapterId != 0 || len(trueDark.PricePhases) != 2 ||
		trueDark.PricePhases[0].PriceId != 1007 || trueDark.PricePhases[1].PriceId != 1007 {
		t.Fatalf("unexpected true-dark chapter catalog entry: %+v", trueDark)
	}
}

func TestChapterGachaPromotionsMatchCoverPickups(t *testing.T) {
	entries := buildChapterGachaEntries()
	EnrichCatalogPromotions(entries, &GachaCatalog{})
	wantChapterMaterials := [...][2]int32{
		{330001, 330009},
		{330005, 330013},
		{330002, 330017},
		{330010, 330006},
		{330014, 330018},
		{330003, 330011},
		{330007, 330015},
		{330023, 330019},
		{330004, 330012},
		{330008, 330016},
		{330020, 330024},
	}

	for i, entry := range entries[:len(wantChapterMaterials)] {
		want := [...]struct {
			id    int32
			count int32
		}{
			{100003, 6},
			{100004, 5},
			{wantChapterMaterials[i][0], 1},
			{wantChapterMaterials[i][1], 1},
		}
		if len(entry.PromotionItems) != len(want) {
			t.Fatalf("chapter %d promotion count = %d, want %d", i+1, len(entry.PromotionItems), len(want))
		}
		for j, pickup := range entry.PromotionItems {
			if pickup.PossessionId != want[j].id || pickup.Count != want[j].count || !pickup.IsTarget {
				t.Errorf("chapter %d promotion %d = %+v, want material %d x%d", i+1, j+1, pickup, want[j].id, want[j].count)
			}
		}
	}
	if got := entries[len(entries)-1].PromotionItems; len(got) != 0 {
		t.Fatalf("true-dark chapter promotions = %+v, want none without configured rewards", got)
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

func TestLoadGachaCatalogDoesNotSynthesizeEventBoxInventory(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog()
	if err != nil {
		t.Fatal(err)
	}
	hasMamaBanner := false
	for _, entry := range entries {
		if entry.GachaLabelType == model.GachaLabelEvent {
			t.Fatalf("event gacha %d was exposed without authoritative inventory", entry.GachaId)
		}
		hasMamaBanner = hasMamaBanner || entry.IsMamaBanner
	}
	if len(entries) == 0 {
		t.Fatal("non-event gachas were removed with event gachas")
	}
	if !hasMamaBanner {
		t.Fatal("m_mom_banner entries were not marked as Mama banners")
	}
}

func TestLoadGachaCatalogIncludesGuaranteedTicketGachas(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog()
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
