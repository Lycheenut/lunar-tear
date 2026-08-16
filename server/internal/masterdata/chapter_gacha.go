package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

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
}

func buildChapterGachaEntries() []store.GachaCatalogEntry {
	entries := make([]store.GachaCatalogEntry, 0, len(chapterGachaSpecs))
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
	return entries
}

func chapterGachaPrerequisiteMainQuestChapterId(mainQuestChapterId int32) int32 {
	for i, spec := range chapterGachaSpecs {
		if spec.mainQuestChapterId != mainQuestChapterId {
			continue
		}
		if i == 0 {
			return 0
		}
		return chapterGachaSpecs[i-1].mainQuestChapterId
	}
	return 0
}
