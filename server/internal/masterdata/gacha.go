package masterdata

import (
	"fmt"
	"sort"
	"strings"

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
	guaranteedThreeStarOrHigherGachaAssetName = "confirm_sr"
	guaranteedFourStarGachaAssetName          = "confirm_ssr"
)

func LoadGachaCatalog() ([]store.GachaCatalogEntry, map[int32]GachaMedalInfo, error) {
	medals, err := utils.ReadTable[EntityMGachaMedal]("m_gacha_medal")
	if err != nil {
		return nil, nil, fmt.Errorf("load gacha medal table: %w", err)
	}
	banners, err := utils.ReadTable[EntityMMomBanner]("m_mom_banner")
	if err != nil {
		return nil, nil, fmt.Errorf("load mom banner table: %w", err)
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
	// Keep the linked ids out of the catalog instead of treating UI display items
	// as drawable stock with fabricated counts and rarity.
	eventGachaIds := make(map[int32]bool)
	for _, chapter := range eventChapters {
		link := linkById[chapter.EventQuestLinkId]
		if link.DestinationDomainType == model.MomBannerDomainGacha {
			eventGachaIds[link.DestinationDomainId] = true
		}
	}

	gachaToMedal := make(map[int32]EntityMGachaMedal)
	medalInfoByGacha := make(map[int32]GachaMedalInfo)
	for _, m := range medals {
		gachaToMedal[m.ShopTransitionGachaId] = m
		medalInfoByGacha[m.ShopTransitionGachaId] = GachaMedalInfo{
			GachaMedalId:        m.GachaMedalId,
			ConsumableItemId:    m.ConsumableItemId,
			AutoConvertDatetime: m.AutoConvertDatetime,
			ConversionRate:      m.ConversionRate,
		}
	}

	stepupSteps := make(map[int32][]EntityMMomBanner)
	var entries []store.GachaCatalogEntry

	for _, b := range banners {
		if b.DestinationDomainType != model.MomBannerDomainGacha {
			continue
		}
		// The snapshot's common_* rows are incomplete proxy banners. Chapter
		// Gachas are reconstructed below from their chapter-specific data.
		if strings.HasPrefix(b.BannerAssetName, model.BannerPrefixCommon) {
			continue
		}
		gachaId := b.DestinationDomainId
		if eventGachaIds[gachaId] {
			continue
		}

		if strings.HasPrefix(b.BannerAssetName, model.BannerPrefixStepUp) {
			if _, hasMedal := gachaToMedal[gachaId]; !hasMedal {
				continue
			}
			groupId := gachaId / model.StepUpGroupDivisor
			stepupSteps[groupId] = append(stepupSteps[groupId], b)
			continue
		}

		decoration := model.GachaDecorationNormal

		if strings.HasPrefix(b.BannerAssetName, model.BannerPrefixLimited) {
			decoration = model.GachaDecorationFestival
		}
		medal, hasMedal := gachaToMedal[gachaId]
		if !hasMedal {
			continue
		}

		entries = append(entries, store.GachaCatalogEntry{
			GachaId:               gachaId,
			IsMamaBanner:          true,
			GachaLabelType:        model.GachaLabelPremium,
			GachaModeType:         model.GachaModeBasic,
			GachaAutoResetType:    model.GachaAutoResetNone,
			IsUserGachaUnlock:     true,
			StartDatetime:         b.StartDatetime,
			EndDatetime:           b.EndDatetime,
			GachaMedalId:          medal.GachaMedalId,
			MedalConsumableItemId: medal.ConsumableItemId,
			GachaDecorationType:   decoration,
			SortOrder:             b.SortOrderDesc,
			BannerAssetName:       b.BannerAssetName,
			GroupId:               gachaId,
			CeilingCount:          model.PityCeilingCount,
			PricePhases:           buildPremiumBasicPricePhases(gachaId),
		})
	}
	entries = append(entries, buildChapterGachaEntries()...)
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
	seenGacha := make(map[int32]bool, len(entries))
	for _, entry := range entries {
		seenGacha[entry.GachaId] = true
	}
	for _, steps := range stepupSteps {
		first := steps[0]
		gachaId := first.DestinationDomainId
		if seenGacha[gachaId] {
			continue
		}

		medal := gachaToMedal[first.DestinationDomainId]
		medalId := medal.GachaMedalId
		medalConsumableId := medal.ConsumableItemId

		pricePhases := buildStepUpPricePhases(gachaId, len(steps))

		var maxStep int32
		for _, p := range pricePhases {
			if p.StepNumber > maxStep {
				maxStep = p.StepNumber
			}
		}

		entries = append(entries, store.GachaCatalogEntry{
			GachaId:               gachaId,
			IsMamaBanner:          true,
			GachaLabelType:        model.GachaLabelPremium,
			GachaModeType:         model.GachaModeStepup,
			GachaAutoResetType:    model.GachaAutoResetNone,
			IsUserGachaUnlock:     true,
			StartDatetime:         first.StartDatetime,
			EndDatetime:           first.EndDatetime,
			GachaMedalId:          medalId,
			MedalConsumableItemId: medalConsumableId,
			GachaDecorationType:   model.GachaDecorationFestival,
			SortOrder:             first.SortOrderDesc,
			BannerAssetName:       first.BannerAssetName,
			GroupId:               gachaId,
			CeilingCount:          model.PityCeilingCount,
			PricePhases:           pricePhases,
			MaxStepNumber:         maxStep,
		})
		seenGacha[gachaId] = true
	}

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
			mainQuestChapterId = chapterGachaPrerequisiteMainQuestChapterId(mainQuestChapterId)
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
			entries[i].PromotionItems = buildChapterGachaPromotionItems(entries[i])
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

func buildPremiumBasicPricePhases(gachaId int32) []store.GachaPricePhaseEntry {
	return []store.GachaPricePhaseEntry{
		{
			PhaseId:      gachaId*model.PhaseIdMultiplier + 1,
			PriceType:    model.PriceTypeGem,
			Price:        0,
			RegularPrice: 0,
			DrawCount:    1,
		},
		{
			PhaseId:        gachaId*model.PhaseIdMultiplier + 2,
			PriceType:      model.PriceTypeGem,
			Price:          0,
			RegularPrice:   0,
			DrawCount:      model.PremiumMultiPullCount,
			FixedRarityMin: model.RaritySRare,
			FixedCount:     1,
		},
		{
			PhaseId:      gachaId*model.PhaseIdMultiplier + 3,
			PriceType:    model.PriceTypeConsumableItem,
			PriceId:      model.ConsumableIdPremiumTicket,
			Price:        0,
			RegularPrice: 0,
			DrawCount:    1,
		},
	}
}

func buildStepUpPricePhases(gachaId int32, totalSteps int) []store.GachaPricePhaseEntry {
	stepCosts := []int32{model.StepUpStep1Cost, model.StepUpFreeCost, model.StepUpStep3Cost, model.StepUpFreeCost, model.StepUpStep5Cost}
	stepCosts = stepCosts[:min(totalSteps, len(stepCosts))]

	var phases []store.GachaPricePhaseEntry
	for i, cost := range stepCosts {
		step := int32(i + 1)
		priceType := model.PriceTypePaidGem
		if cost == 0 {
			priceType = model.PriceTypeGem
		}

		fixedRarityMin := int32(0)
		fixedCount := int32(0)
		if step == int32(len(stepCosts)) {
			fixedRarityMin = model.RaritySSRare
			fixedCount = 1
		}

		phases = append(phases, store.GachaPricePhaseEntry{
			PhaseId:        gachaId*model.PhaseIdMultiplier + step,
			PriceType:      priceType,
			Price:          cost,
			RegularPrice:   0,
			DrawCount:      model.PremiumMultiPullCount,
			FixedRarityMin: fixedRarityMin,
			FixedCount:     fixedCount,
			LimitExecCount: 1,
			StepNumber:     step,
		})
	}
	return phases
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
