package masterdata

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

type GachaMedalInfo struct {
	GachaMedalId        int32
	ConsumableItemId    int32
	AutoConvertDatetime int64
	ConversionRate      int32
}

const chapterGachaIdBase int32 = 200000

// These ticket-only Gachas have no m_mom_banner rows in this snapshot.
const (
	dailyGachaAssetName                       = "daily_1"
	guaranteedThreeStarOrHigherGachaAssetName = "confirm_sr"
	guaranteedFourStarGachaAssetName          = "confirm_ssr"
)

func LoadGachaCatalog(dailyUnlockQuestId int32) ([]store.GachaCatalogEntry, map[int32]GachaMedalInfo, error) {
	medals, err := utils.ReadTable[EntityMGachaMedal]("m_gacha_medal")
	if err != nil {
		return nil, nil, fmt.Errorf("load gacha medal table: %w", err)
	}
	eventChapters, err := utils.ReadTable[EntityMEventQuestChapter]("m_event_quest_chapter")
	if err != nil {
		return nil, nil, fmt.Errorf("load event quest chapter table: %w", err)
	}
	eventLinks, err := utils.ReadTable[EntityMEventQuestLink]("m_event_quest_link")
	if err != nil {
		return nil, nil, fmt.Errorf("load event quest link table: %w", err)
	}
	linkById := make(map[int32]EntityMEventQuestLink)
	for _, link := range eventLinks {
		linkById[link.EventQuestLinkId] = link
	}
	// This master-data snapshot has no authoritative event-box inventory table.
	// Event metadata still comes from the quest link; the separately managed
	// Gacha JSON supplies the boxes and rewards.
	eventGachaChapters := make(map[int32]EntityMEventQuestChapter)
	eventGachaLinks := make(map[int32]EntityMEventQuestLink)
	for _, chapter := range eventChapters {
		link := linkById[chapter.EventQuestLinkId]
		if link.DestinationDomainType == model.MomBannerDomainGacha {
			eventGachaChapters[link.DestinationDomainId] = chapter
			eventGachaLinks[link.DestinationDomainId] = link
		}
	}

	medalInfoByGacha := make(map[int32]GachaMedalInfo)
	for _, m := range medals {
		medalInfoByGacha[m.ShopTransitionGachaId] = GachaMedalInfo{
			GachaMedalId:        m.GachaMedalId,
			ConsumableItemId:    m.ConsumableItemId,
			AutoConvertDatetime: m.AutoConvertDatetime,
			ConversionRate:      m.ConversionRate,
		}
	}

	entries := buildChapterGachaEntries()
	entries = append(entries, buildEventGachaEntries(eventGachaChapters, eventGachaLinks)...)
	if dailyUnlockQuestId > 0 {
		entries = append(entries, buildDailyGacha(dailyUnlockQuestId))
	}
	entries = append(entries,
		buildGuaranteedTicketGacha(
			model.GachaIdGuaranteedThreeStarOrHigher,
			model.ConsumableIdGuaranteedThreeStarOrHigherTicket,
			guaranteedThreeStarOrHigherGachaAssetName,
			model.RaritySRare,
		),
		buildGuaranteedTicketGacha(
			model.GachaIdGuaranteedFourStar,
			model.ConsumableIdGuaranteedFourStarTicket,
			guaranteedFourStarGachaAssetName,
			model.RaritySSRare,
		),
	)
	// The client opens the first active Gacha after sorting ascending. Keep the
	// synthetic ticket pools behind every master-data-backed pool and give them
	// the catalog's permanent availability window instead of the Unix epoch.
	maxSortOrder := int32(0)
	var catalogStartDatetime int64
	var catalogEndDatetime int64
	for _, entry := range entries {
		if model.IsGuaranteedTicketGacha(entry.GachaId) {
			continue
		}
		if entry.SortOrder > maxSortOrder {
			maxSortOrder = entry.SortOrder
		}
		if entry.StartDatetime > 0 && (catalogStartDatetime == 0 || entry.StartDatetime < catalogStartDatetime) {
			catalogStartDatetime = entry.StartDatetime
		}
		if entry.EndDatetime > catalogEndDatetime {
			catalogEndDatetime = entry.EndDatetime
		}
	}
	for i := range entries {
		if model.IsGuaranteedTicketGacha(entries[i].GachaId) {
			maxSortOrder++
			entries[i].SortOrder = maxSortOrder
			entries[i].StartDatetime = catalogStartDatetime
			entries[i].EndDatetime = catalogEndDatetime
		}
	}

	return entries, medalInfoByGacha, nil
}

func buildEventGachaEntries(chapters map[int32]EntityMEventQuestChapter, links map[int32]EntityMEventQuestLink) []store.GachaCatalogEntry {
	gachaIds := make([]int32, 0, len(chapters))
	for gachaId := range chapters {
		gachaIds = append(gachaIds, gachaId)
	}
	sort.Slice(gachaIds, func(i, j int) bool { return gachaIds[i] < gachaIds[j] })

	entries := make([]store.GachaCatalogEntry, 0, len(gachaIds))
	for _, gachaId := range gachaIds {
		chapter := chapters[gachaId]
		link := links[gachaId]
		if link.PossessionType != int32(model.PossessionTypeConsumableItem) || link.PossessionId <= 0 {
			continue
		}
		entries = append(entries, store.GachaCatalogEntry{
			GachaId:                    gachaId,
			IsMamaBanner:               true,
			GachaLabelType:             model.GachaLabelEvent,
			GachaModeType:              model.GachaModeBox,
			GachaAutoResetType:         model.GachaAutoResetNone,
			IsUserGachaUnlock:          true,
			RequiredConsumableItemId:   link.PossessionId,
			StartDatetime:              chapter.StartDatetime,
			EndDatetime:                chapter.EndDatetime,
			RelatedEventQuestChapterId: chapter.EventQuestChapterId,
			GachaDecorationType:        model.GachaDecorationNormal,
			SortOrder:                  chapter.DisplaySortOrder,
			BannerAssetName:            fmt.Sprintf("event_%d", gachaId),
			GroupId:                    gachaId,
			PricePhases:                buildChapterPricePhases(gachaId, link.PossessionId),
			DescriptionTextId:          chapter.NameEventQuestTextId,
		})
	}
	return entries
}

func buildDailyGacha(unlockQuestId int32) store.GachaCatalogEntry {
	return store.GachaCatalogEntry{
		GachaId:              model.GachaIdDaily,
		GachaLabelType:       model.GachaLabelPremium,
		GachaModeType:        model.GachaModeBasic,
		GachaAutoResetType:   model.GachaAutoResetDaily,
		GachaAutoResetPeriod: 1,
		IsUserGachaUnlock:    true,
		GachaDecorationType:  model.GachaDecorationNormal,
		SortOrder:            model.GachaIdDaily,
		BannerAssetName:      dailyGachaAssetName,
		GroupId:              model.GachaIdDaily,
		UnlockConditions: []store.GachaUnlockConditionEntry{{
			GachaUnlockConditionType: model.GachaUnlockMainQuestClear,
			ConditionValue:           unlockQuestId,
		}},
		PricePhases: []store.GachaPricePhaseEntry{{
			PhaseId:        model.GachaIdDaily*model.PhaseIdMultiplier + 1,
			PriceType:      model.PriceTypeGem,
			DrawCount:      model.DailyGachaDrawCount,
			LimitExecCount: model.DailyGachaExecLimit,
		}},
	}
}

func buildGuaranteedTicketGacha(gachaId, ticketId int32, assetName string, minimumRarity model.RarityType) store.GachaCatalogEntry {
	return store.GachaCatalogEntry{
		GachaId:                  gachaId,
		GachaLabelType:           model.GachaLabelPremium,
		GachaModeType:            model.GachaModeBasic,
		GachaAutoResetType:       model.GachaAutoResetNone,
		IsUserGachaUnlock:        true,
		RequiredConsumableItemId: ticketId,
		GachaDecorationType:      model.GachaDecorationNormal,
		BannerAssetName:          assetName,
		GroupId:                  gachaId,
		PricePhases: []store.GachaPricePhaseEntry{{
			PhaseId:        gachaId*model.PhaseIdMultiplier + 1,
			PriceType:      model.PriceTypeConsumableItem,
			PriceId:        ticketId,
			Price:          1,
			DrawCount:      1,
			FixedRarityMin: minimumRarity,
			FixedCount:     1,
		}},
	}
}

func EnrichGachaUnlockConditions(entries []store.GachaCatalogEntry, quests *QuestCatalog) {
	lastQuestByChapter := make(map[int32]int32)
	for _, questId := range quests.OrderedQuestIds {
		if chapterId := quests.MainQuestChapterIdByQuestId[questId]; chapterId != 0 {
			lastQuestByChapter[chapterId] = questId
		}
	}
	for i := range entries {
		mainQuestChapterId := entries[i].RelatedMainQuestChapterId
		if entries[i].GachaLabelType == model.GachaLabelChapter {
			mainQuestChapterId = chapterGachaPrerequisiteMainQuestChapterId(entries, i, quests)
		}
		if mainQuestChapterId != 0 {
			if questId := lastQuestByChapter[mainQuestChapterId]; questId != 0 {
				entries[i].UnlockConditions = []store.GachaUnlockConditionEntry{{GachaUnlockConditionType: model.GachaUnlockMainQuestClear, ConditionValue: questId}}
			}
		}
		if entries[i].RelatedEventQuestChapterId != 0 {
			for _, questId := range quests.EventUnlockQuestIdsForChapter(entries[i].RelatedEventQuestChapterId) {
				entries[i].UnlockConditions = append(entries[i].UnlockConditions, store.GachaUnlockConditionEntry{GachaUnlockConditionType: model.GachaUnlockMainQuestClear, ConditionValue: questId})
			}
		}
		if len(entries[i].UnlockConditions) == 0 {
			entries[i].UnlockConditions = []store.GachaUnlockConditionEntry{{GachaUnlockConditionType: model.GachaUnlockNone}}
		}
	}
}

const chapterPromoMaxItems = 4

func EnrichCatalogPromotions(entries []store.GachaCatalogEntry, pool *GachaCatalog) {
	for i := range entries {
		if entries[i].GachaLabelType == model.GachaLabelEvent {
			entries[i].PromotionItems = buildBoxPromotionItems(entries[i].BoxItems)
			continue
		}
		if entries[i].GachaLabelType == model.GachaLabelChapter {
			entries[i].PromotionItems = nil
			continue
		}

		featured := pool.FeaturedByGacha[entries[i].GachaId]
		items := make([]store.GachaPromotionItem, 0, len(featured.Costumes)+len(featured.Weapons))
		for _, c := range featured.Costumes {
			items = append(items, toPromoItemWithBonus(c, pool))
		}
		for _, w := range featured.Weapons {
			items = append(items, toPromoItemWithBonus(w, pool))
		}

		entries[i].PromotionItems = items
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].SortOrder != entries[j].SortOrder {
			return entries[i].SortOrder < entries[j].SortOrder
		}
		return entries[i].GachaId < entries[j].GachaId
	})
}

func toPromoItem(item GachaPoolItem) store.GachaPromotionItem {
	return store.GachaPromotionItem{
		PossessionType: item.PossessionType,
		PossessionId:   item.PossessionId,
		IsTarget:       true,
	}
}

func toPromoItemWithBonus(item GachaPoolItem, pool *GachaCatalog) store.GachaPromotionItem {
	pi := store.GachaPromotionItem{
		PossessionType: item.PossessionType,
		PossessionId:   item.PossessionId,
		IsTarget:       true,
	}
	if item.PossessionType == int32(model.PossessionTypeCostume) {
		pi.BonusPossessionType = int32(model.PossessionTypeWeapon)
		pi.BonusPossessionId = pool.CostumeWeaponMap[item.PossessionId]
	}
	return pi
}

func buildBoxPromotionItems(boxItems []store.GachaBoxItemEntry) []store.GachaPromotionItem {
	limit := min(chapterPromoMaxItems, len(boxItems))
	items := make([]store.GachaPromotionItem, 0, limit)
	for _, item := range boxItems[:limit] {
		items = append(items, store.GachaPromotionItem{
			PossessionType:   item.PossessionType,
			PossessionId:     item.PossessionId,
			Count:            item.Count,
			MaxDrawableCount: item.MaxCount,
			CounterId:        item.CounterId,
			IsTarget:         true,
		})
	}
	return items
}

func buildChapterPricePhases(gachaId, ticketId int32) []store.GachaPricePhaseEntry {
	return []store.GachaPricePhaseEntry{
		{
			PhaseId:      gachaId*model.PhaseIdMultiplier + 1,
			PriceType:    model.PriceTypeConsumableItem,
			PriceId:      ticketId,
			Price:        1,
			RegularPrice: 0,
			DrawCount:    1,
		},
		{
			PhaseId:      gachaId*model.PhaseIdMultiplier + 2,
			PriceType:    model.PriceTypeConsumableItem,
			PriceId:      ticketId,
			Price:        10,
			RegularPrice: 0,
			DrawCount:    model.PremiumMultiPullCount,
		},
	}
}
