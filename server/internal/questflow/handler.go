package questflow

import (
	"sort"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/importantitem"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questdrop"
	"lunar-tear/server/internal/store"
)

type RewardGrant struct {
	PossessionType model.PossessionType
	PossessionId   int32
	Count          int32
	RewardEffectId int32
	IsAutoSale     bool
}

type FinishOutcome struct {
	DropRewards                  []RewardGrant
	FirstClearRewards            []RewardGrant
	ReplayFlowFirstClearRewards  []RewardGrant
	MissionClearRewards          []RewardGrant
	MissionClearCompleteRewards  []RewardGrant
	BigWinClearedQuestMissionIds []int32
	IsBigWin                     bool
	ChangedWeaponStoryIds        []int32
	ClearedQuestMissionIds       []int32
	ReplayRewardGroupId          int32
}

type QuestHandler struct {
	*masterdata.QuestCatalog
	Config                         *masterdata.GameConfig
	Granter                        *store.PossessionGranter
	SideStoryChapterByEventQuestId map[int32]int32
	Campaigns                      *campaign.Catalog
	ImportantItemEffects           *importantitem.Catalog
	CharacterRebirth               *masterdata.CharacterRebirthCatalog
	DropRewardsByQuestID           map[int32][]questdrop.Reward
}

func NewQuestHandler(catalog *masterdata.QuestCatalog, config *masterdata.GameConfig, sideStory *masterdata.SideStoryCatalog, campaigns *campaign.Catalog, importantItemEffects *importantitem.Catalog, characterRebirth *masterdata.CharacterRebirthCatalog) *QuestHandler {
	granter := BuildGranter(catalog, config)
	var sideStoryChapters map[int32]int32
	if sideStory != nil {
		sideStoryChapters = sideStory.ChapterByEventQuestId
	}
	return &QuestHandler{
		QuestCatalog:                   catalog,
		Config:                         config,
		Granter:                        granter,
		SideStoryChapterByEventQuestId: sideStoryChapters,
		Campaigns:                      campaigns,
		ImportantItemEffects:           importantItemEffects,
		CharacterRebirth:               characterRebirth,
	}
}

func BuildGranter(catalog *masterdata.QuestCatalog, config *masterdata.GameConfig) *store.PossessionGranter {
	costumeById := make(map[int32]store.CostumeRef, len(catalog.CostumeById))
	for id, cm := range catalog.CostumeById {
		costumeById[id] = store.CostumeRef{CharacterId: cm.CharacterId}
	}
	costumeEnhancedById := make(map[int32]store.CostumeEnhancedRef, len(catalog.CostumeEnhancedById))
	for id, enhanced := range catalog.CostumeEnhancedById {
		var exp int32
		if costume, ok := catalog.CostumeById[enhanced.CostumeId]; ok {
			thresholds := catalog.CostumeExpByRarity[costume.RarityType]
			if enhanced.Level >= 0 && int(enhanced.Level) < len(thresholds) {
				exp = thresholds[enhanced.Level]
			}
		}
		costumeEnhancedById[id] = store.CostumeEnhancedRef{
			CostumeId: enhanced.CostumeId,
			Level:     enhanced.Level,
			Exp:       exp,
		}
	}
	weaponById := make(map[int32]store.WeaponRef, len(catalog.WeaponById))
	for id, wm := range catalog.WeaponById {
		weaponById[id] = store.WeaponRef{
			WeaponSkillGroupId:                 wm.WeaponSkillGroupId,
			WeaponAbilityGroupId:               wm.WeaponAbilityGroupId,
			WeaponStoryReleaseConditionGroupId: wm.WeaponStoryReleaseConditionGroupId,
		}
	}
	releaseConditions := make(map[int32][]store.WeaponStoryReleaseCond, len(catalog.ReleaseConditionsByGroupId))
	for groupId, rows := range catalog.ReleaseConditionsByGroupId {
		conds := make([]store.WeaponStoryReleaseCond, len(rows))
		for i, r := range rows {
			conds[i] = store.WeaponStoryReleaseCond{
				StoryIndex:                      r.StoryIndex,
				WeaponStoryReleaseConditionType: model.WeaponStoryReleaseConditionType(r.WeaponStoryReleaseConditionType),
				ConditionValue:                  r.ConditionValue,
			}
		}
		releaseConditions[groupId] = conds
	}
	partsById := make(map[int32]store.PartsRef, len(catalog.PartsById))
	partsVariants := make(map[int32]map[int32][]int32)
	for id, p := range catalog.PartsById {
		partsById[id] = store.PartsRef{
			PartsGroupId:                  p.PartsGroupId,
			RarityType:                    p.RarityType,
			PartsInitialLotteryId:         p.PartsInitialLotteryId,
			PartsStatusMainLotteryGroupId: p.PartsStatusMainLotteryGroupId,
			PartsStatusSubLotteryGroupId:  p.PartsStatusSubLotteryGroupId,
		}
		if partsVariants[p.PartsGroupId] == nil {
			partsVariants[p.PartsGroupId] = map[int32][]int32{}
		}
		partsVariants[p.PartsGroupId][p.RarityType] = append(partsVariants[p.PartsGroupId][p.RarityType], p.PartsId)
	}
	for _, byRarity := range partsVariants {
		for _, ids := range byRarity {
			sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		}
	}

	partsSubDefs := make(map[int32]store.PartsStatusSubDef, len(catalog.PartsStatusMainById))
	for id, d := range catalog.PartsStatusMainById {
		var fn func(int32) int32
		if f, ok := catalog.FuncResolver.Resolve(d.StatusNumericalFunctionId); ok {
			fn = f.Evaluate
		}
		partsSubDefs[id] = store.PartsStatusSubDef{
			StatusKindType:           d.StatusKindType,
			StatusCalculationType:    d.StatusCalculationType,
			StatusChangeInitialValue: d.StatusChangeInitialValue,
			StatusFunc:               fn,
		}
	}

	partsSellPriceL1 := make(map[int32]int32, len(catalog.SellPriceByRarity))
	for rarity, fn := range catalog.SellPriceByRarity {
		partsSellPriceL1[int32(rarity)] = fn.Evaluate(1)
	}
	var goldItemId int32
	if config != nil {
		goldItemId = config.ConsumableItemIdForGold
	}

	return &store.PossessionGranter{
		CostumeById:                          costumeById,
		CostumeEnhancedById:                  costumeEnhancedById,
		WeaponById:                           weaponById,
		WeaponSkillSlots:                     catalog.WeaponSkillSlots,
		WeaponAbilitySlots:                   catalog.WeaponAbilitySlots,
		ReleaseConditions:                    releaseConditions,
		PartsById:                            partsById,
		DefaultPartsStatusMainByLotteryGroup: catalog.DefaultPartsStatusMainByLotteryGroup,
		PartsVariantsByGroupRarity:           partsVariants,
		PartsSubStatusPool:                   catalog.SubStatusPool,
		PartsSubStatusDefs:                   partsSubDefs,
		PartsSellPriceL1ByRarity:             partsSellPriceL1,
		GoldConsumableItemId:                 goldItemId,
	}
}
