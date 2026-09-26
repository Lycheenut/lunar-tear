package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

const trueDarkMemoryQuestChapterId int32 = 901

type chapterGachaSpec struct {
	mainQuestChapterId int32
	ticketId           int32
}

var chapterGachaSpecs = []chapterGachaSpec{
	{2, 1008},
	{3, 1009},
	{4, 1010},
	{5, 1011},
	{6, 1012},
	{7, 1013},
	{9, 1014},
	{10, 1015},
	{11, 1016},
	{12, 1017},
	{13, 1018},
	{14, 1019},
	{17, 1020},
	{18, 1021},
	{19, 1022},
	{20, 1023},
	{21, 1024},
	{22, 1025},
	{25, 1026},
	{26, 1027},
	{27, 1028},
	{28, 1029},
	{29, 1030},
	{30, 1031},
}

func buildChapterGachaEntries() []store.GachaCatalogEntry {
	entries := make([]store.GachaCatalogEntry, 0, len(chapterGachaSpecs)+1)
	for i, spec := range chapterGachaSpecs {
		chapterNumber := int32(i + 1)
		gachaId := chapterGachaIdBase + chapterNumber
		entries = append(entries, store.GachaCatalogEntry{
			GachaId:                   gachaId,
			IsMamaBanner:              true,
			GachaLabelType:            model.GachaLabelChapter,
			GachaModeType:             model.GachaModeBox,
			GachaAutoResetType:        model.GachaAutoResetMonthly,
			GachaAutoResetPeriod:      1,
			IsUserGachaUnlock:         true,
			RelatedMainQuestChapterId: spec.mainQuestChapterId,
			GachaDecorationType:       model.GachaDecorationNormal,
			SortOrder:                 chapterNumber,
			BannerAssetName:           fmt.Sprintf("chapter_%d", chapterNumber),
			GroupId:                   gachaId,
			PricePhases:               buildChapterPricePhases(gachaId, spec.ticketId),
			DescriptionTextId:         gachaId,
		})
	}
	entries = append(entries, store.GachaCatalogEntry{
		GachaId:                    chapterGachaIdBase,
		IsMamaBanner:               true,
		GachaLabelType:             model.GachaLabelChapter,
		GachaModeType:              model.GachaModeBox,
		GachaAutoResetType:         model.GachaAutoResetMonthly,
		GachaAutoResetPeriod:       1,
		IsUserGachaUnlock:          true,
		RelatedEventQuestChapterId: trueDarkMemoryQuestChapterId,
		GachaDecorationType:        model.GachaDecorationNormal,
		SortOrder:                  int32(len(chapterGachaSpecs) + 1),
		BannerAssetName:            "chapter_ex",
		GroupId:                    chapterGachaIdBase,
		PricePhases:                buildChapterPricePhases(chapterGachaIdBase, model.ConsumableIdDarkMemorySummonTicket),
		DescriptionTextId:          chapterGachaIdBase,
	})
	return entries
}

func chapterGachaPrerequisiteMainQuestChapterId(
	entries []store.GachaCatalogEntry,
	entryIndex int,
	quests *QuestCatalog,
) int32 {
	if entryIndex < 0 || entryIndex >= len(entries) {
		return 0
	}
	chapterId := entries[entryIndex].RelatedMainQuestChapterId
	if chapterId == 0 {
		return 0
	}
	routeId := quests.MainQuestRouteIdByChapterId[chapterId]
	seasonId := quests.SeasonIdByRouteId[routeId]
	var fallbackChapterId int32
	var previousSeasonChapterId int32
	for i := entryIndex - 1; i >= 0; i-- {
		previous := entries[i]
		if previous.GachaLabelType != model.GachaLabelChapter || previous.RelatedMainQuestChapterId == 0 {
			continue
		}
		if fallbackChapterId == 0 {
			fallbackChapterId = previous.RelatedMainQuestChapterId
		}
		if routeId == 0 {
			return fallbackChapterId
		}
		previousRouteId := quests.MainQuestRouteIdByChapterId[previous.RelatedMainQuestChapterId]
		if previousRouteId == routeId {
			return previous.RelatedMainQuestChapterId
		}
		previousSeasonId := quests.SeasonIdByRouteId[previousRouteId]
		if previousSeasonChapterId == 0 && previousSeasonId > 0 && previousSeasonId < seasonId {
			previousSeasonChapterId = previous.RelatedMainQuestChapterId
		}
	}
	if previousSeasonChapterId != 0 {
		return previousSeasonChapterId
	}
	return fallbackChapterId
}
