package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type chapterWeightedItem struct {
	possessionType int32
	possessionId   int32
	count          int32
	monthlyLimit   int32
	weight         int32
}

type chapterGachaProfile struct {
	consumables     []chapterWeightedItem
	characterWeight [6]int32
	skillWeight     [6]int32
	goldWeight      int32
}

type chapterGachaSpec struct {
	mainQuestChapterId     int32
	ticketId               int32
	limitedCharacterWeight [3]int32
	companions             []chapterWeightedItem
	profile                chapterGachaProfile
	rareSkillMaterialId    int32
	baseSkillMaterialId    int32
}

var earlyChapterGachaProfile = chapterGachaProfile{
	consumables: []chapterWeightedItem{
		consumableChapterItem(3003, 1, 1, 10),
		consumableChapterItem(3002, 1, 3, 30),
		consumableChapterItem(3001, 1, 5, 50),
		consumableChapterItem(2001, 1, 1, 10),
	},
	characterWeight: [6]int32{300, 200, 400, 300, 500, 400},
	skillWeight:     [6]int32{300, 600, 900, 600, 800, 1200},
	goldWeight:      500,
}

var chapterFourGachaProfile = chapterGachaProfile{
	consumables: []chapterWeightedItem{
		consumableChapterItem(3003, 1, 1, 10),
		consumableChapterItem(3002, 1, 3, 20),
		consumableChapterItem(3001, 1, 5, 50),
		consumableChapterItem(2001, 1, 1, 10),
	},
	characterWeight: [6]int32{230, 150, 310, 230, 380, 310},
	skillWeight:     [6]int32{310, 460, 620, 770, 920, 1150},
	goldWeight:      1150,
}

var middleChapterGachaProfile = chapterGachaProfile{
	consumables:     earlyChapterGachaProfile.consumables,
	characterWeight: chapterFourGachaProfile.characterWeight,
	skillWeight:     chapterFourGachaProfile.skillWeight,
	goldWeight:      chapterFourGachaProfile.goldWeight,
}

var chapterSevenEightGachaProfile = chapterGachaProfile{
	consumables: []chapterWeightedItem{
		consumableChapterItem(3003, 1, 1, 8),
		consumableChapterItem(3002, 1, 3, 23),
		consumableChapterItem(3001, 1, 5, 38),
		consumableChapterItem(2001, 1, 1, 8),
	},
	characterWeight: [6]int32{231, 154, 308, 231, 385, 308},
	skillWeight:     [6]int32{769, 923, 1154, 308, 462, 615},
	goldWeight:      1154,
}

var lateChapterGachaProfile = chapterGachaProfile{
	consumables: []chapterWeightedItem{
		consumableChapterItem(2001, 1, 1, 10),
		consumableChapterItem(3003, 2, 5, 60),
		consumableChapterItem(3002, 2, 15, 190),
		consumableChapterItem(3001, 2, 25, 320),
	},
	characterWeight: earlyChapterGachaProfile.characterWeight,
	skillWeight:     earlyChapterGachaProfile.skillWeight,
	goldWeight:      earlyChapterGachaProfile.goldWeight,
}

var chapterGachaSpecs = []chapterGachaSpec{
	{2, 1008, [3]int32{200, 300, 400}, []chapterWeightedItem{materialChapterItem(330001, 1, 100, 1000), materialChapterItem(330009, 1, 100, 990)}, earlyChapterGachaProfile, 301010, 301009},
	{3, 1009, [3]int32{190, 300, 410}, []chapterWeightedItem{materialChapterItem(330005, 1, 100, 970), materialChapterItem(330013, 1, 100, 1020)}, earlyChapterGachaProfile, 301002, 301001},
	{4, 1010, [3]int32{190, 300, 400}, []chapterWeightedItem{materialChapterItem(330002, 1, 100, 990), materialChapterItem(330017, 1, 100, 1020)}, earlyChapterGachaProfile, 301012, 301011},
	{5, 1011, [3]int32{210, 310, 380}, []chapterWeightedItem{materialChapterItem(330010, 1, 100, 1010), materialChapterItem(330006, 1, 100, 1000)}, chapterFourGachaProfile, 301004, 301003},
	{6, 1012, [3]int32{200, 300, 400}, []chapterWeightedItem{materialChapterItem(330014, 1, 100, 1000), materialChapterItem(330018, 1, 100, 1000)}, middleChapterGachaProfile, 301006, 301005},
	{7, 1013, [3]int32{210, 340, 420}, []chapterWeightedItem{materialChapterItem(330011, 1, 100, 960), materialChapterItem(330003, 1, 100, 980)}, middleChapterGachaProfile, 301008, 301007},
	{9, 1014, [3]int32{151, 227, 287}, []chapterWeightedItem{materialChapterItem(330007, 1, 100, 756), materialChapterItem(330015, 1, 100, 748), materialChapterItem(330021, 1, 100, 756)}, chapterSevenEightGachaProfile, 301010, 301009},
	{10, 1015, [3]int32{151, 219, 295}, []chapterWeightedItem{materialChapterItem(330023, 1, 100, 748), materialChapterItem(330022, 1, 100, 756), materialChapterItem(330019, 1, 100, 756)}, chapterSevenEightGachaProfile, 301002, 301001},
	{11, 1016, [3]int32{250, 380, 510}, []chapterWeightedItem{materialChapterItem(330004, 1, 50, 640), materialChapterItem(330012, 1, 50, 640)}, lateChapterGachaProfile, 301012, 301011},
	{12, 1017, [3]int32{250, 380, 510}, []chapterWeightedItem{materialChapterItem(330008, 1, 50, 640), materialChapterItem(330016, 1, 50, 640)}, lateChapterGachaProfile, 301004, 301003},
	{13, 1018, [3]int32{250, 380, 510}, []chapterWeightedItem{materialChapterItem(330020, 1, 50, 640), materialChapterItem(330024, 1, 50, 640)}, lateChapterGachaProfile, 301006, 301005},
}

func materialChapterItem(id, count, limit, weight int32) chapterWeightedItem {
	return chapterWeightedItem{possessionType: int32(model.PossessionTypeMaterial), possessionId: id, count: count, monthlyLimit: limit, weight: weight}
}

func consumableChapterItem(id, count, limit, weight int32) chapterWeightedItem {
	return chapterWeightedItem{possessionType: int32(model.PossessionTypeConsumableItem), possessionId: id, count: count, monthlyLimit: limit, weight: weight}
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
			BoxItems:                  buildChapterGachaItems(spec),
			DescriptionTextId:         gachaId,
		})
	}
	return entries
}

func buildChapterGachaItems(spec chapterGachaSpec) []store.GachaBoxItemEntry {
	items := make([]chapterWeightedItem, 0, 23)
	for i, count := range [...]int32{5, 4, 3} {
		items = append(items, materialChapterItem(100004, count, [...]int32{20, 30, 40}[i], spec.limitedCharacterWeight[i]))
	}
	items = append(items, spec.companions...)
	items = append(items, spec.profile.consumables...)

	for _, item := range []chapterWeightedItem{
		materialChapterItem(100003, 6, 0, spec.profile.characterWeight[0]),
		materialChapterItem(100003, 4, 0, spec.profile.characterWeight[1]),
		materialChapterItem(100002, 8, 0, spec.profile.characterWeight[2]),
		materialChapterItem(100002, 6, 0, spec.profile.characterWeight[3]),
		materialChapterItem(100001, 10, 0, spec.profile.characterWeight[4]),
		materialChapterItem(100001, 8, 0, spec.profile.characterWeight[5]),
	} {
		items = append(items, item)
	}

	for i, count := range [...]int32{4, 3, 2} {
		items = append(items, materialChapterItem(spec.rareSkillMaterialId, count, 0, spec.profile.skillWeight[i]))
	}
	for i, count := range [...]int32{6, 5, 4} {
		items = append(items, materialChapterItem(spec.baseSkillMaterialId, count, 0, spec.profile.skillWeight[i+3]))
	}
	items = append(items, consumableChapterItem(1, 5, 0, spec.profile.goldWeight))

	result := make([]store.GachaBoxItemEntry, 0, len(items))
	for i, item := range items {
		result = append(result, store.GachaBoxItemEntry{
			PossessionType: item.possessionType,
			PossessionId:   item.possessionId,
			RarityType:     chapterGachaItemRarity(item.possessionType, item.possessionId),
			Count:          item.count,
			MaxCount:       item.monthlyLimit,
			CounterId:      int32(i + 1),
			Weight:         item.weight,
		})
	}
	return result
}

func chapterGachaItemRarity(possessionType, possessionId int32) int32 {
	if possessionType == int32(model.PossessionTypeConsumableItem) {
		return int32(model.RarityNormal)
	}
	switch possessionId {
	case 100001, 330001, 330005, 330009, 330013, 330017, 330021:
		return int32(model.RarityNormal)
	case 100002, 301001, 301003, 301005, 301007, 301009, 301011,
		330002, 330006, 330010, 330014, 330018, 330022:
		return int32(model.RarityRare)
	default:
		return int32(model.RaritySRare)
	}
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
