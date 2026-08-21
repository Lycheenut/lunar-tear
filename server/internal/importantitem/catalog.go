package importantitem

import (
	"fmt"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/utils"
)

type effectType int32

const (
	effectTypeUnlockFunction effectType = 1
	effectTypeDropRate       effectType = 2
	effectTypeDropCount      effectType = 3
)

type questTargetType int32

const (
	questTargetWholeQuest          questTargetType = 1
	questTargetQuestType           questTargetType = 2
	questTargetEventQuestType      questTargetType = 3
	questTargetMainQuestChapter    questTargetType = 4
	questTargetMainQuest           questTargetType = 5
	questTargetEventQuestChapter   questTargetType = 6
	questTargetEventQuest          questTargetType = 7
	questTargetMainQuestDifficulty questTargetType = 8
)

type questTarget struct {
	typeId questTargetType
	value  int32
}

type possessionKey struct {
	typeId model.PossessionType
	id     int32
}

type effectRule struct {
	importantItemId int32
	typeId          effectType
	bonusPermil     int32
	questTargets    []questTarget
	itemTargets     map[possessionKey]struct{}
	startMillis     int64
	endMillis       int64
}

type Catalog struct {
	rules []effectRule
}

func Load() (*Catalog, error) {
	items, err := utils.ReadTable[masterdata.EntityMImportantItem]("m_important_item")
	if err != nil {
		return nil, fmt.Errorf("load important item table: %w", err)
	}
	effects, err := utils.ReadTable[masterdata.EntityMImportantItemEffect]("m_important_item_effect")
	if err != nil {
		return nil, fmt.Errorf("load important item effect table: %w", err)
	}
	dropCounts, err := utils.ReadTable[masterdata.EntityMImportantItemEffectDropCount]("m_important_item_effect_drop_count")
	if err != nil {
		return nil, fmt.Errorf("load important item drop-count effect table: %w", err)
	}
	dropRates, err := utils.ReadTable[masterdata.EntityMImportantItemEffectDropRate]("m_important_item_effect_drop_rate")
	if err != nil {
		return nil, fmt.Errorf("load important item drop-rate effect table: %w", err)
	}
	questTargets, err := utils.ReadTable[masterdata.EntityMImportantItemEffectTargetQuestGroup]("m_important_item_effect_target_quest_group")
	if err != nil {
		return nil, fmt.Errorf("load important item quest-target table: %w", err)
	}
	itemTargets, err := utils.ReadTable[masterdata.EntityMImportantItemEffectTargetItemGroup]("m_important_item_effect_target_item_group")
	if err != nil {
		return nil, fmt.Errorf("load important item possession-target table: %w", err)
	}

	effectById := make(map[int32]masterdata.EntityMImportantItemEffect, len(effects))
	for _, effect := range effects {
		effectById[effect.ImportantItemEffectId] = effect
	}
	dropCountById := make(map[int32]masterdata.EntityMImportantItemEffectDropCount, len(dropCounts))
	for _, effect := range dropCounts {
		dropCountById[effect.ImportantItemEffectDropCountId] = effect
	}
	dropRateById := make(map[int32]masterdata.EntityMImportantItemEffectDropRate, len(dropRates))
	for _, effect := range dropRates {
		dropRateById[effect.ImportantItemEffectDropRateId] = effect
	}
	questTargetsByGroup := make(map[int32][]questTarget)
	for _, target := range questTargets {
		questTargetsByGroup[target.ImportantItemEffectTargetQuestGroupId] = append(
			questTargetsByGroup[target.ImportantItemEffectTargetQuestGroupId],
			questTarget{typeId: questTargetType(target.ImportantItemEffectTargetQuestGroupType), value: target.TargetValue},
		)
	}
	itemTargetsByGroup := make(map[int32]map[possessionKey]struct{})
	for _, target := range itemTargets {
		group := itemTargetsByGroup[target.ImportantItemEffectTargetItemGroupId]
		if group == nil {
			group = make(map[possessionKey]struct{})
			itemTargetsByGroup[target.ImportantItemEffectTargetItemGroupId] = group
		}
		group[possessionKey{typeId: model.PossessionType(target.PossessionType), id: target.PossessionId}] = struct{}{}
	}

	catalog := &Catalog{}
	for _, item := range items {
		if item.ImportantItemEffectId == 0 {
			continue
		}
		effect, ok := effectById[item.ImportantItemEffectId]
		if !ok {
			return nil, fmt.Errorf("important item %d references missing effect %d", item.ImportantItemId, item.ImportantItemEffectId)
		}
		if effectType(effect.ImportantItemEffectType) == effectTypeUnlockFunction {
			continue
		}

		rule := effectRule{
			importantItemId: item.ImportantItemId,
			typeId:          effectType(effect.ImportantItemEffectType),
			startMillis:     effect.StartDatetime,
			endMillis:       effect.EndDatetime,
		}
		var questGroupId, itemGroupId int32
		switch rule.typeId {
		case effectTypeDropRate:
			definition, ok := dropRateById[effect.ImportantItemEffectTargetId]
			if !ok {
				return nil, fmt.Errorf("important item effect %d references missing drop-rate effect %d", effect.ImportantItemEffectId, effect.ImportantItemEffectTargetId)
			}
			rule.bonusPermil = definition.RatePermil
			questGroupId = definition.ImportantItemEffectTargetQuestGroupId
			itemGroupId = definition.ImportantItemEffectTargetItemGroupId
		case effectTypeDropCount:
			definition, ok := dropCountById[effect.ImportantItemEffectTargetId]
			if !ok {
				return nil, fmt.Errorf("important item effect %d references missing drop-count effect %d", effect.ImportantItemEffectId, effect.ImportantItemEffectTargetId)
			}
			rule.bonusPermil = definition.CountPermil
			questGroupId = definition.ImportantItemEffectTargetQuestGroupId
			itemGroupId = definition.ImportantItemEffectTargetItemGroupId
		default:
			return nil, fmt.Errorf("important item effect %d has unsupported type %d", effect.ImportantItemEffectId, effect.ImportantItemEffectType)
		}

		rule.questTargets = questTargetsByGroup[questGroupId]
		if len(rule.questTargets) == 0 {
			return nil, fmt.Errorf("important item effect %d references empty quest-target group %d", effect.ImportantItemEffectId, questGroupId)
		}
		if itemGroupId != 0 {
			rule.itemTargets = itemTargetsByGroup[itemGroupId]
			if len(rule.itemTargets) == 0 {
				return nil, fmt.Errorf("important item effect %d references empty possession-target group %d", effect.ImportantItemEffectId, itemGroupId)
			}
		}
		catalog.rules = append(catalog.rules, rule)
	}
	return catalog, nil
}

func (c *Catalog) EffectCount() int {
	if c == nil {
		return 0
	}
	return len(c.rules)
}

func (c *Catalog) QuestBonuses(
	importantItems map[int32]int32,
	target campaign.QuestTarget,
	possessionType model.PossessionType,
	possessionId int32,
	nowMillis int64,
) (dropRatePermil, dropCountPermil int32) {
	if c == nil {
		return 0, 0
	}
	key := possessionKey{typeId: possessionType, id: possessionId}
	for _, rule := range c.rules {
		if importantItems[rule.importantItemId] <= 0 || nowMillis < rule.startMillis || nowMillis > rule.endMillis {
			continue
		}
		if !matchesQuest(rule.questTargets, target) {
			continue
		}
		if len(rule.itemTargets) > 0 {
			if _, ok := rule.itemTargets[key]; !ok {
				continue
			}
		}
		switch rule.typeId {
		case effectTypeDropRate:
			dropRatePermil += rule.bonusPermil
		case effectTypeDropCount:
			dropCountPermil += rule.bonusPermil
		}
	}
	return dropRatePermil, dropCountPermil
}

func matchesQuest(targets []questTarget, quest campaign.QuestTarget) bool {
	for _, target := range targets {
		switch target.typeId {
		case questTargetWholeQuest:
			return true
		case questTargetQuestType:
			if int32(quest.QuestType) == target.value {
				return true
			}
		case questTargetEventQuestType:
			if quest.QuestType == campaign.QuestTypeEventQuest && quest.EventQuestType == target.value {
				return true
			}
		case questTargetMainQuestChapter:
			if quest.QuestType == campaign.QuestTypeMainQuest && quest.ChapterId == target.value {
				return true
			}
		case questTargetMainQuest:
			if quest.QuestType == campaign.QuestTypeMainQuest && quest.QuestId == target.value {
				return true
			}
		case questTargetEventQuestChapter:
			if quest.QuestType == campaign.QuestTypeEventQuest && quest.ChapterId == target.value {
				return true
			}
		case questTargetEventQuest:
			if quest.QuestType == campaign.QuestTypeEventQuest && quest.QuestId == target.value {
				return true
			}
		case questTargetMainQuestDifficulty:
			if quest.QuestType == campaign.QuestTypeMainQuest && quest.MainQuestDifficultyType == target.value {
				return true
			}
		}
	}
	return false
}
