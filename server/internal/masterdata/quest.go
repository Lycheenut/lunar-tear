package masterdata

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/utils"
)

const (
	battleDropEffectNormal int32 = 1
	battleDropEffectRare   int32 = 2
	battleDropEffectHigh   int32 = 3

	consumableItemTypeGachaTicket   int32 = 200
	consumableItemTypeExploreTicket int32 = 201
	consumableItemTypeQuestSkip     int32 = 202
	consumableItemTypeEffect        int32 = 300
)

type BattleDropInfo struct {
	QuestSceneId         int32
	BattleDropCategoryId int32
	BattleDropEffectId   int32
	BattleDropRewardId   int32
}

type WeaponEvolutionInfo struct {
	GroupId int32
	Order   int32
}

func battleDropEffectId(
	reward EntityMBattleDropReward,
	materialRarity map[int32]int32,
	partsById map[int32]EntityMParts,
	consumableType map[int32]int32,
) int32 {
	var rarity int32
	switch model.PossessionType(reward.PossessionType) {
	case model.PossessionTypeMaterial:
		rarity = materialRarity[reward.PossessionId]
	case model.PossessionTypeParts, model.PossessionTypePartsEnhanced:
		rarity = partsById[reward.PossessionId].RarityType
	case model.PossessionTypeConsumableItem:
		switch consumableType[reward.PossessionId] {
		case consumableItemTypeGachaTicket, consumableItemTypeExploreTicket,
			consumableItemTypeQuestSkip, consumableItemTypeEffect:
			return battleDropEffectHigh
		default:
			return battleDropEffectNormal
		}
	case model.PossessionTypeFreeGem, model.PossessionTypePaidGem:
		return battleDropEffectHigh
	}

	switch {
	case rarity >= model.RaritySRare:
		return battleDropEffectHigh
	case rarity >= model.RarityRare:
		return battleDropEffectRare
	default:
		return battleDropEffectNormal
	}
}

type EventQuestDailyGroup struct {
	Definition EntityMEventQuestDailyGroup
	ChapterIds []int32
	Rewards    []RewardItem
}

type EventQuestUnlockCondition struct {
	EventQuestType  int32
	CharacterId     int32
	QuestId         int32
	RequiredQuestId int32
}

type QuestSceneChoiceKey struct {
	QuestSceneId  int32
	QuestFlowType int32
	ChoiceNumber  int32
}

type QuestCatalog struct {
	SceneById                          map[int32]EntityMQuestScene
	MissionById                        map[int32]EntityMQuestMission
	QuestById                          map[int32]EntityMQuest
	MissionIdsByQuestId                map[int32][]int32
	RouteIdByQuestId                   map[int32]int32
	MainQuestDifficultyTypeByQuestId   map[int32]int32
	SceneIdsByQuestId                  map[int32][]int32
	OrderedQuestIds                    []int32
	FirstClearRewardsByGroupId         map[int32][]EntityMQuestFirstClearRewardGroup
	FirstClearRewardSwitchesByQuestId  map[int32][]EntityMQuestFirstClearRewardSwitch
	MissionRewardsByMissionId          map[int32][]EntityMQuestMissionReward
	MissionConditionValuesByGroupId    map[int32][]int32
	SceneChoiceByKey                   map[QuestSceneChoiceKey]EntityMQuestSceneChoice
	WeaponIdsByReleaseConditionGroupId map[int32][]int32
	ReleaseConditionsByGroupId         map[int32][]EntityMWeaponStoryReleaseConditionGroup
	SceneGrantsBySceneId               map[int32][]EntityMUserQuestSceneGrantPossession
	BattleDropRewardById               map[int32]EntityMBattleDropReward
	PickupRewardIdsByGroupId           map[int32][]int32
	PickupRewardIdsByGroupAndEffectId  map[int32]map[int32][]int32
	BattleDropEffectIdByRewardId       map[int32]int32
	BattleDropsByQuestId               map[int32][]BattleDropInfo
	BossCountByQuestId                 map[int32]int32
	QuestBonusById                     map[int32]EntityMQuestBonus
	QuestBonusCharacterRowsByGroupId   map[int32][]EntityMQuestBonusCharacterGroup
	QuestBonusWeaponRowsByGroupId      map[int32][]EntityMQuestBonusWeaponGroup
	QuestBonusEffectsByGroupId         map[int32][]EntityMQuestBonusEffectGroup
	QuestBonusDropByEffectId           map[int32]EntityMQuestBonusDropReward
	QuestBonusExpByEffectId            map[int32]EntityMQuestBonusExp
	QuestBonusTermsByGroupId           map[int32][]EntityMQuestBonusTermGroup
	WeaponEvolutionByWeaponId          map[int32]WeaponEvolutionInfo
	ReplayFlowRewardsByGroupId         map[int32][]EntityMQuestReplayFlowRewardGroup
	RentalQuestIds                     map[int32]bool
	TutorialUnlockConditions           []EntityMTutorialUnlockCondition
	ChapterLastSceneByQuestId          map[int32]int32
	SeasonIdByRouteId                  map[int32]int32
	RoutesBySeason                     map[int32][]int32
	RouteCompletionQuestId             map[int32]int32
	BattleOnlyTargetSceneByQuestId     map[int32]int32
	MainQuestChapterIdByQuestId        map[int32]int32
	MainQuestRouteIdByChapterId        map[int32]int32
	EventQuestTypeByChapterId          map[int32]int32
	EventChapterById                   map[int32]EntityMEventQuestChapter
	EventQuestIdsByChapterId           map[int32][]int32
	EventQuestIdsByChapterSortOrder    map[int32]map[int32][]int32
	EventQuestIdsByChapterDifficulty   map[int32]map[int32][]int32
	LimitContentQuestIds               map[int32]bool
	EventUnlockConditions              []EventQuestUnlockCondition
	EventCharacterIdsByChapterId       map[int32]map[int32]bool
	EventDailyGroups                   []EventQuestDailyGroup

	UserExpThresholds       []int32
	CharacterExpThresholds  []int32
	CostumeExpByRarity      map[int32][]int32
	CostumeMaxLevelByRarity map[int32]NumericalFunc
	MaxStaminaByLevel       map[int32]int32

	CostumeById         map[int32]EntityMCostume
	CostumeEnhancedById map[int32]EntityMCostumeEnhanced
	WeaponById          map[int32]EntityMWeapon

	WeaponSkillSlots   map[int32][]int32
	WeaponAbilitySlots map[int32][]int32

	*PartsCatalog
}

func (c *QuestCatalog) EventQuestBelongsToChapter(chapterId, questId int32) bool {
	for _, candidateId := range c.EventQuestIdsByChapterId[chapterId] {
		if candidateId == questId {
			return true
		}
	}
	return false
}

func (c *QuestCatalog) EventUnlockQuestIdsForChapter(chapterId int32) []int32 {
	chapter, ok := c.EventChapterById[chapterId]
	if !ok {
		return nil
	}
	chapterQuests := make(map[int32]bool, len(c.EventQuestIdsByChapterId[chapterId]))
	for _, questId := range c.EventQuestIdsByChapterId[chapterId] {
		chapterQuests[questId] = true
	}
	seen := make(map[int32]bool)
	var result []int32
	for _, condition := range c.EventUnlockConditions {
		if condition.EventQuestType != chapter.EventQuestType ||
			(condition.CharacterId != 0 && !c.EventCharacterIdsByChapterId[chapterId][condition.CharacterId]) ||
			(condition.QuestId != 0 && !chapterQuests[condition.QuestId]) || seen[condition.RequiredQuestId] {
			continue
		}
		seen[condition.RequiredQuestId] = true
		result = append(result, condition.RequiredQuestId)
	}
	return result
}

func buildEventQuestIndexes(
	chapters []EntityMEventQuestChapter,
	groups []EntityMEventQuestSequenceGroup,
	sequences []EntityMEventQuestSequence,
) (map[int32][]int32, map[int32]map[int32][]int32, map[int32]map[int32][]int32) {
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].EventQuestSequenceGroupId != groups[j].EventQuestSequenceGroupId {
			return groups[i].EventQuestSequenceGroupId < groups[j].EventQuestSequenceGroupId
		}
		if groups[i].DifficultyType != groups[j].DifficultyType {
			return groups[i].DifficultyType < groups[j].DifficultyType
		}
		return groups[i].EventQuestSequenceId < groups[j].EventQuestSequenceId
	})
	sort.Slice(sequences, func(i, j int) bool {
		if sequences[i].EventQuestSequenceId != sequences[j].EventQuestSequenceId {
			return sequences[i].EventQuestSequenceId < sequences[j].EventQuestSequenceId
		}
		if sequences[i].SortOrder != sequences[j].SortOrder {
			return sequences[i].SortOrder < sequences[j].SortOrder
		}
		return sequences[i].QuestId < sequences[j].QuestId
	})

	type sequenceRef struct {
		id         int32
		difficulty int32
	}
	sequencesByGroup := make(map[int32][]sequenceRef)
	for _, row := range groups {
		sequencesByGroup[row.EventQuestSequenceGroupId] = append(sequencesByGroup[row.EventQuestSequenceGroupId], sequenceRef{
			id: row.EventQuestSequenceId, difficulty: row.DifficultyType,
		})
	}
	questRowsBySequence := make(map[int32][]EntityMEventQuestSequence)
	for _, row := range sequences {
		questRowsBySequence[row.EventQuestSequenceId] = append(questRowsBySequence[row.EventQuestSequenceId], row)
	}

	questIdsByChapter := make(map[int32][]int32)
	questIdsByChapterSortOrder := make(map[int32]map[int32][]int32)
	questIdsByChapterDifficulty := make(map[int32]map[int32][]int32)
	for _, chapter := range chapters {
		seen := make(map[int32]bool)
		seenBySortOrder := make(map[int32]map[int32]bool)
		bySortOrder := make(map[int32][]int32)
		seenByDifficulty := make(map[int32]map[int32]bool)
		byDifficulty := make(map[int32][]int32)
		for _, sequence := range sequencesByGroup[chapter.EventQuestSequenceGroupId] {
			for _, row := range questRowsBySequence[sequence.id] {
				if !seen[row.QuestId] {
					seen[row.QuestId] = true
					questIdsByChapter[chapter.EventQuestChapterId] = append(questIdsByChapter[chapter.EventQuestChapterId], row.QuestId)
				}
				if seenByDifficulty[sequence.difficulty] == nil {
					seenByDifficulty[sequence.difficulty] = make(map[int32]bool)
				}
				if !seenByDifficulty[sequence.difficulty][row.QuestId] {
					seenByDifficulty[sequence.difficulty][row.QuestId] = true
					byDifficulty[sequence.difficulty] = append(byDifficulty[sequence.difficulty], row.QuestId)
				}
				if seenBySortOrder[row.SortOrder] == nil {
					seenBySortOrder[row.SortOrder] = make(map[int32]bool)
				}
				if !seenBySortOrder[row.SortOrder][row.QuestId] {
					seenBySortOrder[row.SortOrder][row.QuestId] = true
					bySortOrder[row.SortOrder] = append(bySortOrder[row.SortOrder], row.QuestId)
				}
			}
		}
		questIdsByChapterSortOrder[chapter.EventQuestChapterId] = bySortOrder
		questIdsByChapterDifficulty[chapter.EventQuestChapterId] = byDifficulty
	}
	return questIdsByChapter, questIdsByChapterSortOrder, questIdsByChapterDifficulty
}

func buildBossCountByQuestId(
	scenes []EntityMQuestScene,
	sceneBattles []EntityMQuestSceneBattle,
	battleGroups []EntityMBattleGroup,
	battles []EntityMBattle,
	npcDecks []EntityMBattleNpcDeck,
	npcCharacterTypes []EntityMBattleNpcDeckCharacterType,
) map[int32]int32 {
	const battleEnemyTypeBoss int32 = 2

	type npcDeckKey struct {
		battleNpcId int64
		deckType    int32
		deckNumber  int32
	}
	type npcCharacterKey struct {
		battleNpcId int64
		uuid        string
	}

	questIdBySceneId := make(map[int32]int32, len(scenes))
	for _, scene := range scenes {
		questIdBySceneId[scene.QuestSceneId] = scene.QuestId
	}
	battleIdsByGroupId := make(map[int32][]int32)
	for _, group := range battleGroups {
		battleIdsByGroupId[group.BattleGroupId] = append(battleIdsByGroupId[group.BattleGroupId], group.BattleId)
	}
	battleById := make(map[int32]EntityMBattle, len(battles))
	for _, battle := range battles {
		battleById[battle.BattleId] = battle
	}
	deckByKey := make(map[npcDeckKey]EntityMBattleNpcDeck, len(npcDecks))
	for _, deck := range npcDecks {
		deckByKey[npcDeckKey{deck.BattleNpcId, deck.DeckType, deck.BattleNpcDeckNumber}] = deck
	}
	enemyTypeByCharacter := make(map[npcCharacterKey]int32, len(npcCharacterTypes))
	for _, characterType := range npcCharacterTypes {
		enemyTypeByCharacter[npcCharacterKey{characterType.BattleNpcId, characterType.BattleNpcDeckCharacterUuid}] = characterType.BattleEnemyType
	}

	counts := make(map[int32]int32)
	for _, sceneBattle := range sceneBattles {
		questId := questIdBySceneId[sceneBattle.QuestSceneId]
		if questId == 0 {
			continue
		}
		for _, battleId := range battleIdsByGroupId[sceneBattle.BattleGroupId] {
			battle, ok := battleById[battleId]
			if !ok {
				continue
			}
			deck, ok := deckByKey[npcDeckKey{battle.BattleNpcId, battle.DeckType, battle.BattleNpcDeckNumber}]
			if !ok {
				continue
			}
			for _, uuid := range []string{deck.BattleNpcDeckCharacterUuid01, deck.BattleNpcDeckCharacterUuid02, deck.BattleNpcDeckCharacterUuid03} {
				if uuid != "" && enemyTypeByCharacter[npcCharacterKey{battle.BattleNpcId, uuid}] == battleEnemyTypeBoss {
					counts[questId]++
				}
			}
		}
	}
	return counts
}

func LoadQuestCatalog(partsCatalog *PartsCatalog, conditionResolver *ConditionResolver) (*QuestCatalog, error) {
	scenes, err := utils.ReadTable[EntityMQuestScene]("m_quest_scene")
	if err != nil {
		return nil, fmt.Errorf("load quest scene table: %w", err)
	}
	sort.Slice(scenes, func(i, j int) bool {
		if scenes[i].QuestId != scenes[j].QuestId {
			return scenes[i].QuestId < scenes[j].QuestId
		}
		if scenes[i].SortOrder != scenes[j].SortOrder {
			return scenes[i].SortOrder < scenes[j].SortOrder
		}
		return scenes[i].QuestSceneId < scenes[j].QuestSceneId
	})

	missions, err := utils.ReadTable[EntityMQuestMission]("m_quest_mission")
	if err != nil {
		return nil, fmt.Errorf("load quest mission table: %w", err)
	}
	missionConditionValues, err := utils.ReadTable[EntityMQuestMissionConditionValueGroup]("m_quest_mission_condition_value_group")
	if err != nil {
		return nil, fmt.Errorf("load quest mission condition values: %w", err)
	}
	sceneChoices, err := utils.ReadTable[EntityMQuestSceneChoice]("m_quest_scene_choice")
	if err != nil {
		return nil, fmt.Errorf("load quest scene choices: %w", err)
	}

	quests, err := utils.ReadTable[EntityMQuest]("m_quest")
	if err != nil {
		return nil, fmt.Errorf("load quest table: %w", err)
	}
	questBonuses, err := utils.ReadTable[EntityMQuestBonus]("m_quest_bonus")
	if err != nil {
		return nil, fmt.Errorf("load quest bonus table: %w", err)
	}
	questBonusCharacters, err := utils.ReadTable[EntityMQuestBonusCharacterGroup]("m_quest_bonus_character_group")
	if err != nil {
		return nil, fmt.Errorf("load quest bonus character table: %w", err)
	}
	questBonusWeapons, err := utils.ReadTable[EntityMQuestBonusWeaponGroup]("m_quest_bonus_weapon_group")
	if err != nil {
		return nil, fmt.Errorf("load quest bonus weapon table: %w", err)
	}
	questBonusEffects, err := utils.ReadTable[EntityMQuestBonusEffectGroup]("m_quest_bonus_effect_group")
	if err != nil {
		return nil, fmt.Errorf("load quest bonus effect table: %w", err)
	}
	questBonusDrops, err := utils.ReadTable[EntityMQuestBonusDropReward]("m_quest_bonus_drop_reward")
	if err != nil {
		return nil, fmt.Errorf("load quest bonus drop reward table: %w", err)
	}
	questBonusExps, err := utils.ReadTable[EntityMQuestBonusExp]("m_quest_bonus_exp")
	if err != nil {
		return nil, fmt.Errorf("load quest bonus exp table: %w", err)
	}
	questBonusTerms, err := utils.ReadTable[EntityMQuestBonusTermGroup]("m_quest_bonus_term_group")
	if err != nil {
		return nil, fmt.Errorf("load quest bonus term table: %w", err)
	}

	missionGroups, err := utils.ReadTable[EntityMQuestMissionGroup]("m_quest_mission_group")
	if err != nil {
		return nil, fmt.Errorf("load quest mission group table: %w", err)
	}
	sort.Slice(missionGroups, func(i, j int) bool {
		if missionGroups[i].QuestMissionGroupId != missionGroups[j].QuestMissionGroupId {
			return missionGroups[i].QuestMissionGroupId < missionGroups[j].QuestMissionGroupId
		}
		if missionGroups[i].SortOrder != missionGroups[j].SortOrder {
			return missionGroups[i].SortOrder < missionGroups[j].SortOrder
		}
		return missionGroups[i].QuestMissionId < missionGroups[j].QuestMissionId
	})

	sequences, err := utils.ReadTable[EntityMMainQuestSequence]("m_main_quest_sequence")
	if err != nil {
		return nil, fmt.Errorf("load main quest sequence table: %w", err)
	}
	sort.Slice(sequences, func(i, j int) bool {
		if sequences[i].MainQuestSequenceId != sequences[j].MainQuestSequenceId {
			return sequences[i].MainQuestSequenceId < sequences[j].MainQuestSequenceId
		}
		if sequences[i].SortOrder != sequences[j].SortOrder {
			return sequences[i].SortOrder < sequences[j].SortOrder
		}
		return sequences[i].QuestId < sequences[j].QuestId
	})
	sequenceGroups, err := utils.ReadTable[EntityMMainQuestSequenceGroup]("m_main_quest_sequence_group")
	if err != nil {
		return nil, fmt.Errorf("load main quest sequence group table: %w", err)
	}

	chapters, err := utils.ReadTable[EntityMMainQuestChapter]("m_main_quest_chapter")
	if err != nil {
		return nil, fmt.Errorf("load main quest chapter table: %w", err)
	}
	mainQuestRouteIdByChapterId := make(map[int32]int32, len(chapters))
	for _, chapter := range chapters {
		mainQuestRouteIdByChapterId[chapter.MainQuestChapterId] = chapter.MainQuestRouteId
	}

	routes, err := utils.ReadTable[EntityMMainQuestRoute]("m_main_quest_route")
	if err != nil {
		return nil, fmt.Errorf("load main quest route table: %w", err)
	}
	seasonIdByRouteId := make(map[int32]int32, len(routes))
	routesBySeason := make(map[int32][]int32, len(routes))
	sortOrderByRoute := make(map[int32]int32, len(routes))
	for _, r := range routes {
		seasonIdByRouteId[r.MainQuestRouteId] = r.MainQuestSeasonId
		routesBySeason[r.MainQuestSeasonId] = append(routesBySeason[r.MainQuestSeasonId], r.MainQuestRouteId)
		sortOrderByRoute[r.MainQuestRouteId] = r.SortOrder
	}
	for seasonId, ids := range routesBySeason {
		s := ids
		sort.Slice(s, func(i, j int) bool { return sortOrderByRoute[s[i]] > sortOrderByRoute[s[j]] })
		routesBySeason[seasonId] = s
	}

	anotherReplayConds, err := utils.ReadTable[EntityMMainQuestRouteAnotherReplayFlowUnlockCondition]("m_main_quest_route_another_replay_flow_unlock_condition")
	if err != nil {
		return nil, fmt.Errorf("load main quest route another replay flow unlock condition table: %w", err)
	}
	evaluateConds, err := utils.ReadTable[EntityMEvaluateCondition]("m_evaluate_condition")
	if err != nil {
		return nil, fmt.Errorf("load evaluate condition table: %w", err)
	}
	valueGroupByConditionId := make(map[int32]int32, len(evaluateConds))
	for _, c := range evaluateConds {
		valueGroupByConditionId[c.EvaluateConditionId] = c.EvaluateConditionValueGroupId
	}
	evaluateValueGroups, err := utils.ReadTable[EntityMEvaluateConditionValueGroup]("m_evaluate_condition_value_group")
	if err != nil {
		return nil, fmt.Errorf("load evaluate condition value group table: %w", err)
	}
	valueByGroupId := make(map[int32]int32, len(evaluateValueGroups))
	for _, vg := range evaluateValueGroups {
		if _, exists := valueByGroupId[vg.EvaluateConditionValueGroupId]; exists {
			continue
		}
		valueByGroupId[vg.EvaluateConditionValueGroupId] = int32(vg.Value)
	}
	routeCompletionQuestId := make(map[int32]int32, len(anotherReplayConds))
	for _, c := range anotherReplayConds {
		valueGroupId, ok := valueGroupByConditionId[c.UnlockEvaluateConditionId]
		if !ok {
			continue
		}
		questId, ok := valueByGroupId[valueGroupId]
		if !ok {
			continue
		}
		routeCompletionQuestId[c.MainQuestRouteId] = questId
	}

	firstClearSwitches, err := utils.ReadTable[EntityMQuestFirstClearRewardSwitch]("m_quest_first_clear_reward_switch")
	if err != nil {
		return nil, fmt.Errorf("load quest first clear reward switch table: %w", err)
	}

	firstClearRewards, err := utils.ReadTable[EntityMQuestFirstClearRewardGroup]("m_quest_first_clear_reward_group")
	if err != nil {
		return nil, fmt.Errorf("load quest first clear reward group table: %w", err)
	}
	sort.Slice(firstClearRewards, func(i, j int) bool {
		if firstClearRewards[i].QuestFirstClearRewardGroupId != firstClearRewards[j].QuestFirstClearRewardGroupId {
			return firstClearRewards[i].QuestFirstClearRewardGroupId < firstClearRewards[j].QuestFirstClearRewardGroupId
		}
		if firstClearRewards[i].SortOrder != firstClearRewards[j].SortOrder {
			return firstClearRewards[i].SortOrder < firstClearRewards[j].SortOrder
		}
		return firstClearRewards[i].QuestFirstClearRewardType < firstClearRewards[j].QuestFirstClearRewardType
	})

	replayFlowRewards, err := utils.ReadTable[EntityMQuestReplayFlowRewardGroup]("m_quest_replay_flow_reward_group")
	if err != nil {
		return nil, fmt.Errorf("load quest replay flow reward group table: %w", err)
	}
	sort.Slice(replayFlowRewards, func(i, j int) bool {
		if replayFlowRewards[i].QuestReplayFlowRewardGroupId != replayFlowRewards[j].QuestReplayFlowRewardGroupId {
			return replayFlowRewards[i].QuestReplayFlowRewardGroupId < replayFlowRewards[j].QuestReplayFlowRewardGroupId
		}
		return replayFlowRewards[i].SortOrder < replayFlowRewards[j].SortOrder
	})

	missionRewards, err := utils.ReadTable[EntityMQuestMissionReward]("m_quest_mission_reward")
	if err != nil {
		return nil, fmt.Errorf("load quest mission reward table: %w", err)
	}

	weapons, err := utils.ReadTable[EntityMWeapon]("m_weapon")
	if err != nil {
		return nil, fmt.Errorf("load weapon table: %w", err)
	}
	weaponEvolutions, err := utils.ReadTable[EntityMWeaponEvolutionGroup]("m_weapon_evolution_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon evolution table: %w", err)
	}

	weaponSkillGroups, err := utils.ReadTable[EntityMWeaponSkillGroup]("m_weapon_skill_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon skill group table: %w", err)
	}

	weaponAbilityGroups, err := utils.ReadTable[EntityMWeaponAbilityGroup]("m_weapon_ability_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon ability group table: %w", err)
	}

	releaseConditions, err := utils.ReadTable[EntityMWeaponStoryReleaseConditionGroup]("m_weapon_story_release_condition_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon story release condition table: %w", err)
	}

	costumeMasters, err := utils.ReadTable[EntityMCostume]("m_costume")
	if err != nil {
		return nil, fmt.Errorf("load costume table: %w", err)
	}
	costumeEnhancedRows, err := utils.ReadTable[EntityMCostumeEnhanced]("m_costume_enhanced")
	if err != nil {
		return nil, fmt.Errorf("load enhanced costume table: %w", err)
	}

	costumeRarities, err := utils.ReadTable[EntityMCostumeRarity]("m_costume_rarity")
	if err != nil {
		return nil, fmt.Errorf("load costume rarity table: %w", err)
	}

	sceneGrants, err := utils.ReadTable[EntityMUserQuestSceneGrantPossession]("m_user_quest_scene_grant_possession")
	if err != nil {
		return nil, fmt.Errorf("load quest scene grant table: %w", err)
	}

	battleDropRewards, err := utils.ReadTable[EntityMBattleDropReward]("m_battle_drop_reward")
	if err != nil {
		return nil, fmt.Errorf("load battle drop reward table: %w", err)
	}
	materials, err := utils.ReadTable[EntityMMaterial]("m_material")
	if err != nil {
		return nil, fmt.Errorf("load material table for battle drops: %w", err)
	}
	consumableItems, err := utils.ReadTable[EntityMConsumableItem]("m_consumable_item")
	if err != nil {
		return nil, fmt.Errorf("load consumable item table for battle drops: %w", err)
	}

	pickupRewardGroups, err := utils.ReadTable[EntityMQuestPickupRewardGroup]("m_quest_pickup_reward_group")
	if err != nil {
		return nil, fmt.Errorf("load quest pickup reward group table: %w", err)
	}
	sort.Slice(pickupRewardGroups, func(i, j int) bool {
		if pickupRewardGroups[i].QuestPickupRewardGroupId != pickupRewardGroups[j].QuestPickupRewardGroupId {
			return pickupRewardGroups[i].QuestPickupRewardGroupId < pickupRewardGroups[j].QuestPickupRewardGroupId
		}
		return pickupRewardGroups[i].SortOrder < pickupRewardGroups[j].SortOrder
	})

	sceneBattles, err := utils.ReadTable[EntityMQuestSceneBattle]("m_quest_scene_battle")
	if err != nil {
		return nil, fmt.Errorf("load quest scene battle table: %w", err)
	}

	battleGroups, err := utils.ReadTable[EntityMBattleGroup]("m_battle_group")
	if err != nil {
		return nil, fmt.Errorf("load battle group table: %w", err)
	}

	battles, err := utils.ReadTable[EntityMBattle]("m_battle")
	if err != nil {
		return nil, fmt.Errorf("load battle table: %w", err)
	}

	npcDecks, err := utils.ReadTable[EntityMBattleNpcDeck]("m_battle_npc_deck")
	if err != nil {
		return nil, fmt.Errorf("load battle npc deck table: %w", err)
	}

	npcDropCategories, err := utils.ReadTable[EntityMBattleNpcDeckCharacterDropCategory]("m_battle_npc_deck_character_drop_category")
	if err != nil {
		return nil, fmt.Errorf("load battle npc drop category table: %w", err)
	}
	npcCharacterTypes, err := utils.ReadTable[EntityMBattleNpcDeckCharacterType]("m_battle_npc_deck_character_type")
	if err != nil {
		return nil, fmt.Errorf("load battle npc character type table: %w", err)
	}

	rentalDecks, err := utils.ReadTable[EntityMBattleRentalDeck]("m_battle_rental_deck")
	if err != nil {
		return nil, fmt.Errorf("load battle rental deck table: %w", err)
	}

	tutorialUnlockConds, err := utils.ReadTable[EntityMTutorialUnlockCondition]("m_tutorial_unlock_condition")
	if err != nil {
		return nil, fmt.Errorf("load tutorial unlock condition table: %w", err)
	}

	battleOnlyTargetSceneByQuestId := make(map[int32]int32)
	for _, scene := range scenes {
		if scene.IsBattleOnlyTarget {
			if _, exists := battleOnlyTargetSceneByQuestId[scene.QuestId]; !exists {
				battleOnlyTargetSceneByQuestId[scene.QuestId] = scene.QuestSceneId
			}
		}
	}

	paramMapRows, err := LoadParameterMap()
	if err != nil {
		return nil, err
	}

	userLevels, err := utils.ReadTable[EntityMUserLevel]("m_user_level")
	if err != nil {
		return nil, fmt.Errorf("load user level table: %w", err)
	}
	maxStaminaByLevel := make(map[int32]int32, len(userLevels))
	for _, ul := range userLevels {
		maxStaminaByLevel[ul.UserLevel] = ul.MaxStamina
	}

	funcResolver, err := LoadFunctionResolver()
	if err != nil {
		return nil, fmt.Errorf("load function resolver: %w", err)
	}

	costumeExpByRarity := make(map[int32][]int32, len(costumeRarities))
	costumeMaxLevelByRarity := make(map[int32]NumericalFunc, len(costumeRarities))
	for _, r := range costumeRarities {
		if _, ok := costumeExpByRarity[r.RarityType]; !ok {
			costumeExpByRarity[r.RarityType] = BuildExpThresholds(paramMapRows, r.RequiredExpForLevelUpNumericalParameterMapId)
		}
		if _, ok := costumeMaxLevelByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.MaxLevelNumericalFunctionId); found {
				costumeMaxLevelByRarity[r.RarityType] = f
			}
		}
	}

	costumeById := make(map[int32]EntityMCostume, len(costumeMasters))
	for _, cm := range costumeMasters {
		costumeById[cm.CostumeId] = cm
	}
	costumeEnhancedById := make(map[int32]EntityMCostumeEnhanced, len(costumeEnhancedRows))
	for _, enhanced := range costumeEnhancedRows {
		costumeEnhancedById[enhanced.CostumeEnhancedId] = enhanced
	}

	weaponById := make(map[int32]EntityMWeapon, len(weapons))
	for _, w := range weapons {
		weaponById[w.WeaponId] = w
	}
	weaponEvolutionByWeaponId := make(map[int32]WeaponEvolutionInfo, len(weaponEvolutions))
	for _, evolution := range weaponEvolutions {
		weaponEvolutionByWeaponId[evolution.WeaponId] = WeaponEvolutionInfo{
			GroupId: evolution.WeaponEvolutionGroupId,
			Order:   evolution.EvolutionOrder,
		}
	}

	skillSlots := make(map[int32][]int32)
	for _, row := range weaponSkillGroups {
		skillSlots[row.WeaponSkillGroupId] = append(skillSlots[row.WeaponSkillGroupId], row.SlotNumber)
	}
	abilitySlots := make(map[int32][]int32)
	for _, row := range weaponAbilityGroups {
		abilitySlots[row.WeaponAbilityGroupId] = append(abilitySlots[row.WeaponAbilityGroupId], row.SlotNumber)
	}

	sceneById := make(map[int32]EntityMQuestScene, len(scenes))
	sceneIdsByQuestId := make(map[int32][]int32)
	for _, scene := range scenes {
		sceneById[scene.QuestSceneId] = scene
		sceneIdsByQuestId[scene.QuestId] = append(sceneIdsByQuestId[scene.QuestId], scene.QuestSceneId)
	}

	missionById := make(map[int32]EntityMQuestMission, len(missions))
	for _, mission := range missions {
		missionById[mission.QuestMissionId] = mission
	}
	missionConditionValuesByGroupId := make(map[int32][]int32)
	for _, row := range missionConditionValues {
		missionConditionValuesByGroupId[row.QuestMissionConditionValueGroupId] = append(
			missionConditionValuesByGroupId[row.QuestMissionConditionValueGroupId], row.ConditionValue)
	}
	sceneChoiceByKey := make(map[QuestSceneChoiceKey]EntityMQuestSceneChoice)
	for _, row := range sceneChoices {
		sceneChoiceByKey[QuestSceneChoiceKey{QuestSceneId: row.MainFlowQuestSceneId, QuestFlowType: row.QuestFlowType, ChoiceNumber: row.ChoiceNumber}] = row
	}

	questById := make(map[int32]EntityMQuest, len(quests))
	for _, quest := range quests {
		questById[quest.QuestId] = quest
	}
	questBonusById := make(map[int32]EntityMQuestBonus, len(questBonuses))
	for _, bonus := range questBonuses {
		questBonusById[bonus.QuestBonusId] = bonus
	}
	questBonusCharacterRowsByGroupId := make(map[int32][]EntityMQuestBonusCharacterGroup)
	for _, row := range questBonusCharacters {
		questBonusCharacterRowsByGroupId[row.QuestBonusCharacterGroupId] = append(
			questBonusCharacterRowsByGroupId[row.QuestBonusCharacterGroupId], row)
	}
	questBonusWeaponRowsByGroupId := make(map[int32][]EntityMQuestBonusWeaponGroup)
	for _, row := range questBonusWeapons {
		questBonusWeaponRowsByGroupId[row.QuestBonusWeaponGroupId] = append(
			questBonusWeaponRowsByGroupId[row.QuestBonusWeaponGroupId], row)
	}
	questBonusEffectsByGroupId := make(map[int32][]EntityMQuestBonusEffectGroup)
	for _, row := range questBonusEffects {
		questBonusEffectsByGroupId[row.QuestBonusEffectGroupId] = append(
			questBonusEffectsByGroupId[row.QuestBonusEffectGroupId], row)
	}
	for groupId := range questBonusEffectsByGroupId {
		rows := questBonusEffectsByGroupId[groupId]
		sort.Slice(rows, func(i, j int) bool { return rows[i].SortOrder < rows[j].SortOrder })
		questBonusEffectsByGroupId[groupId] = rows
	}
	questBonusDropByEffectId := make(map[int32]EntityMQuestBonusDropReward, len(questBonusDrops))
	for _, row := range questBonusDrops {
		questBonusDropByEffectId[row.QuestBonusEffectId] = row
	}
	questBonusExpByEffectId := make(map[int32]EntityMQuestBonusExp, len(questBonusExps))
	for _, row := range questBonusExps {
		questBonusExpByEffectId[row.QuestBonusEffectId] = row
	}
	questBonusTermsByGroupId := make(map[int32][]EntityMQuestBonusTermGroup)
	for _, row := range questBonusTerms {
		questBonusTermsByGroupId[row.QuestBonusTermGroupId] = append(
			questBonusTermsByGroupId[row.QuestBonusTermGroupId], row)
	}

	missionIdsByGroupId := make(map[int32][]int32, len(missionGroups))
	for _, mg := range missionGroups {
		missionIdsByGroupId[mg.QuestMissionGroupId] = append(
			missionIdsByGroupId[mg.QuestMissionGroupId], mg.QuestMissionId)
	}
	missionIdsByQuestId := make(map[int32][]int32)
	for questId, quest := range questById {
		missionIds := missionIdsByGroupId[quest.QuestMissionGroupId]
		if len(missionIds) == 0 {
			continue
		}
		missionIdsByQuestId[questId] = append([]int32(nil), missionIds...)
	}

	chapterBySequenceGroupId := make(map[int32]EntityMMainQuestChapter, len(chapters))
	for _, chapter := range chapters {
		chapterBySequenceGroupId[chapter.MainQuestSequenceGroupId] = chapter
	}
	chapterBySequenceId := make(map[int32]EntityMMainQuestChapter, len(sequenceGroups))
	difficultyTypeBySequenceId := make(map[int32]int32, len(sequenceGroups))
	for _, group := range sequenceGroups {
		if chapter, ok := chapterBySequenceGroupId[group.MainQuestSequenceGroupId]; ok {
			chapterBySequenceId[group.MainQuestSequenceId] = chapter
			difficultyTypeBySequenceId[group.MainQuestSequenceId] = group.DifficultyType
		}
	}
	routeIdByQuestId := make(map[int32]int32)
	mainQuestChapterIdByQuestId := make(map[int32]int32)
	mainQuestDifficultyTypeByQuestId := make(map[int32]int32)
	for _, sequence := range sequences {
		if chapter, ok := chapterBySequenceId[sequence.MainQuestSequenceId]; ok {
			routeIdByQuestId[sequence.QuestId] = chapter.MainQuestRouteId
			mainQuestChapterIdByQuestId[sequence.QuestId] = chapter.MainQuestChapterId
			mainQuestDifficultyTypeByQuestId[sequence.QuestId] = difficultyTypeBySequenceId[sequence.MainQuestSequenceId]
		}
	}

	eventChapters, err := utils.ReadTable[EntityMEventQuestChapter]("m_event_quest_chapter")
	if err != nil {
		return nil, fmt.Errorf("load event quest chapter table: %w", err)
	}
	eventLimitRelations, err := utils.ReadTable[EntityMEventQuestChapterLimitContentRelation]("m_event_quest_chapter_limit_content_relation")
	if err != nil {
		return nil, fmt.Errorf("load event quest limit content relation table: %w", err)
	}
	eventQuestTypeByChapterId := make(map[int32]int32, len(eventChapters))
	eventChapterById := make(map[int32]EntityMEventQuestChapter, len(eventChapters))
	for _, ec := range eventChapters {
		eventQuestTypeByChapterId[ec.EventQuestChapterId] = ec.EventQuestType
		eventChapterById[ec.EventQuestChapterId] = ec
	}
	eventChapterCharacters, err := utils.ReadTable[EntityMEventQuestChapterCharacter]("m_event_quest_chapter_character")
	if err != nil {
		return nil, fmt.Errorf("load event quest chapter characters: %w", err)
	}
	eventCharacterIdsByChapterId := make(map[int32]map[int32]bool)
	for _, row := range eventChapterCharacters {
		if eventCharacterIdsByChapterId[row.EventQuestChapterId] == nil {
			eventCharacterIdsByChapterId[row.EventQuestChapterId] = make(map[int32]bool)
		}
		eventCharacterIdsByChapterId[row.EventQuestChapterId][row.CharacterId] = true
	}
	eventSequenceGroups, err := utils.ReadTable[EntityMEventQuestSequenceGroup]("m_event_quest_sequence_group")
	if err != nil {
		return nil, fmt.Errorf("load event quest sequence groups: %w", err)
	}
	eventSequences, err := utils.ReadTable[EntityMEventQuestSequence]("m_event_quest_sequence")
	if err != nil {
		return nil, fmt.Errorf("load event quest sequences: %w", err)
	}
	eventQuestIdsByChapterId, eventQuestIdsByChapterSortOrder, eventQuestIdsByChapterDifficulty := buildEventQuestIndexes(eventChapters, eventSequenceGroups, eventSequences)
	limitContentQuestIds := make(map[int32]bool)
	for _, relation := range eventLimitRelations {
		for _, questId := range eventQuestIdsByChapterId[relation.EventQuestChapterId] {
			limitContentQuestIds[questId] = true
		}
	}
	eventUnlockRows, err := utils.ReadTable[EntityMEventQuestUnlockCondition]("m_event_quest_unlock_condition")
	if err != nil {
		return nil, fmt.Errorf("load event unlock conditions: %w", err)
	}
	eventUnlockConditions := make([]EventQuestUnlockCondition, 0, len(eventUnlockRows))
	for _, row := range eventUnlockRows {
		var questId int32
		if row.UnlockEvaluateConditionId != 0 {
			var ok bool
			questId, ok = conditionResolver.RequiredQuestId(row.UnlockEvaluateConditionId)
			if !ok {
				return nil, fmt.Errorf("event quest type %d has unsupported evaluate condition %d", row.EventQuestType, row.UnlockEvaluateConditionId)
			}
		} else {
			switch row.UnlockConditionType {
			case 0:
				continue
			case 1:
				questId = row.ConditionValue
			default:
				return nil, fmt.Errorf("event quest type %d has unsupported unlock condition type %d", row.EventQuestType, row.UnlockConditionType)
			}
		}
		if questId == 0 {
			return nil, fmt.Errorf("event quest type %d has an empty unlock quest", row.EventQuestType)
		}
		eventUnlockConditions = append(eventUnlockConditions, EventQuestUnlockCondition{
			EventQuestType:  row.EventQuestType,
			CharacterId:     row.CharacterId,
			QuestId:         row.QuestId,
			RequiredQuestId: questId,
		})
	}
	dailyRows, err := utils.ReadTable[EntityMEventQuestDailyGroup]("m_event_quest_daily_group")
	if err != nil {
		return nil, fmt.Errorf("load event daily groups: %w", err)
	}
	dailyTargets, err := utils.ReadTable[EntityMEventQuestDailyGroupTargetChapter]("m_event_quest_daily_group_target_chapter")
	if err != nil {
		return nil, fmt.Errorf("load event daily targets: %w", err)
	}
	dailyRewards, err := utils.ReadTable[EntityMEventQuestDailyGroupCompleteReward]("m_event_quest_daily_group_complete_reward")
	if err != nil {
		return nil, fmt.Errorf("load event daily rewards: %w", err)
	}
	chaptersByTarget := make(map[int32][]int32)
	for _, row := range dailyTargets {
		chaptersByTarget[row.EventQuestDailyGroupTargetChapterId] = append(chaptersByTarget[row.EventQuestDailyGroupTargetChapterId], row.EventQuestChapterId)
	}
	rewardsByGroup := make(map[int32][]RewardItem)
	for _, row := range dailyRewards {
		rewardsByGroup[row.EventQuestDailyGroupCompleteRewardId] = append(rewardsByGroup[row.EventQuestDailyGroupCompleteRewardId], RewardItem{PossessionType: row.PossessionType, PossessionId: row.PossessionId, Count: row.Count})
	}
	eventDailyGroups := make([]EventQuestDailyGroup, 0, len(dailyRows))
	for _, row := range dailyRows {
		eventDailyGroups = append(eventDailyGroups, EventQuestDailyGroup{Definition: row, ChapterIds: chaptersByTarget[row.EventQuestDailyGroupTargetChapterId], Rewards: rewardsByGroup[row.EventQuestDailyGroupCompleteRewardId]})
	}

	sortedChapters := make([]EntityMMainQuestChapter, len(chapters))
	copy(sortedChapters, chapters)
	sort.Slice(sortedChapters, func(i, j int) bool {
		return sortedChapters[i].SortOrder < sortedChapters[j].SortOrder
	})
	sequencesByGroupId := make(map[int32][]EntityMMainQuestSequence)
	for _, seq := range sequences {
		sequencesByGroupId[seq.MainQuestSequenceId] = append(sequencesByGroupId[seq.MainQuestSequenceId], seq)
	}
	var orderedQuestIds []int32
	for _, chapter := range sortedChapters {
		for _, seq := range sequencesByGroupId[chapter.MainQuestSequenceGroupId] {
			orderedQuestIds = append(orderedQuestIds, seq.QuestId)
		}
	}

	chapterLastSceneByQuestId := make(map[int32]int32)
	for _, chapter := range sortedChapters {
		seqs := sequencesByGroupId[chapter.MainQuestSequenceGroupId]
		var chapterLastScene int32
		for i := len(seqs) - 1; i >= 0; i-- {
			if sids := sceneIdsByQuestId[seqs[i].QuestId]; len(sids) > 0 {
				chapterLastScene = sids[len(sids)-1]
				break
			}
		}
		if chapterLastScene != 0 {
			for _, seq := range seqs {
				chapterLastSceneByQuestId[seq.QuestId] = chapterLastScene
			}
		}
	}

	firstClearRewardsByGroupId := make(map[int32][]EntityMQuestFirstClearRewardGroup, len(firstClearRewards))
	for _, reward := range firstClearRewards {
		firstClearRewardsByGroupId[reward.QuestFirstClearRewardGroupId] = append(
			firstClearRewardsByGroupId[reward.QuestFirstClearRewardGroupId], reward)
	}

	replayFlowRewardsByGroupId := make(map[int32][]EntityMQuestReplayFlowRewardGroup, len(replayFlowRewards))
	for _, reward := range replayFlowRewards {
		replayFlowRewardsByGroupId[reward.QuestReplayFlowRewardGroupId] = append(
			replayFlowRewardsByGroupId[reward.QuestReplayFlowRewardGroupId], reward)
	}

	firstClearRewardSwitchesByQuestId := make(map[int32][]EntityMQuestFirstClearRewardSwitch, len(firstClearSwitches))
	for _, switchRow := range firstClearSwitches {
		firstClearRewardSwitchesByQuestId[switchRow.QuestId] = append(
			firstClearRewardSwitchesByQuestId[switchRow.QuestId], switchRow)
	}

	missionRewardsByMissionId := make(map[int32][]EntityMQuestMissionReward, len(missionRewards))
	for _, reward := range missionRewards {
		missionRewardsByMissionId[reward.QuestMissionRewardId] = append(
			missionRewardsByMissionId[reward.QuestMissionRewardId], reward)
	}

	weaponIdsByReleaseConditionGroupId := make(map[int32][]int32)
	for _, w := range weaponById {
		if w.WeaponStoryReleaseConditionGroupId != 0 {
			weaponIdsByReleaseConditionGroupId[w.WeaponStoryReleaseConditionGroupId] = append(
				weaponIdsByReleaseConditionGroupId[w.WeaponStoryReleaseConditionGroupId], w.WeaponId)
		}
	}

	releaseConditionsByGroupId := make(map[int32][]EntityMWeaponStoryReleaseConditionGroup)
	for _, c := range releaseConditions {
		releaseConditionsByGroupId[c.WeaponStoryReleaseConditionGroupId] = append(
			releaseConditionsByGroupId[c.WeaponStoryReleaseConditionGroupId], c)
	}

	sceneGrantsBySceneId := make(map[int32][]EntityMUserQuestSceneGrantPossession)
	for _, sg := range sceneGrants {
		sceneGrantsBySceneId[sg.QuestSceneId] = append(sceneGrantsBySceneId[sg.QuestSceneId], sg)
	}

	battleDropRewardById := make(map[int32]EntityMBattleDropReward, len(battleDropRewards))
	for _, bdr := range battleDropRewards {
		battleDropRewardById[bdr.BattleDropRewardId] = bdr
	}
	materialRarityById := make(map[int32]int32, len(materials))
	for _, material := range materials {
		materialRarityById[material.MaterialId] = material.RarityType
	}
	consumableTypeById := make(map[int32]int32, len(consumableItems))
	for _, item := range consumableItems {
		consumableTypeById[item.ConsumableItemId] = item.ConsumableItemType
	}
	partsById := map[int32]EntityMParts{}
	if partsCatalog != nil {
		partsById = partsCatalog.PartsById
	}
	battleDropEffectIdByRewardId := make(map[int32]int32, len(battleDropRewards))
	for _, reward := range battleDropRewards {
		battleDropEffectIdByRewardId[reward.BattleDropRewardId] = battleDropEffectId(
			reward, materialRarityById, partsById, consumableTypeById)
	}

	pickupRewardIdsByGroupId := make(map[int32][]int32)
	pickupRewardIdsByGroupAndEffectId := make(map[int32]map[int32][]int32)
	for _, pg := range pickupRewardGroups {
		pickupRewardIdsByGroupId[pg.QuestPickupRewardGroupId] = append(
			pickupRewardIdsByGroupId[pg.QuestPickupRewardGroupId], pg.BattleDropRewardId)
		if pickupRewardIdsByGroupAndEffectId[pg.QuestPickupRewardGroupId] == nil {
			pickupRewardIdsByGroupAndEffectId[pg.QuestPickupRewardGroupId] = make(map[int32][]int32)
		}
		effectId := battleDropEffectIdByRewardId[pg.BattleDropRewardId]
		pickupRewardIdsByGroupAndEffectId[pg.QuestPickupRewardGroupId][effectId] = append(
			pickupRewardIdsByGroupAndEffectId[pg.QuestPickupRewardGroupId][effectId], pg.BattleDropRewardId)
	}

	battleGroupBySceneId := make(map[int32]int32, len(sceneBattles))
	for _, sb := range sceneBattles {
		battleGroupBySceneId[sb.QuestSceneId] = sb.BattleGroupId
	}

	battleRowsByGroupId := make(map[int32][]EntityMBattleGroup)
	for _, bg := range battleGroups {
		battleRowsByGroupId[bg.BattleGroupId] = append(battleRowsByGroupId[bg.BattleGroupId], bg)
	}
	for groupId := range battleRowsByGroupId {
		rows := battleRowsByGroupId[groupId]
		sort.Slice(rows, func(i, j int) bool { return rows[i].WaveNumber < rows[j].WaveNumber })
		battleRowsByGroupId[groupId] = rows
	}

	type npcDeckKey struct {
		BattleNpcId         int64
		DeckType            int32
		BattleNpcDeckNumber int32
	}
	npcDeckByKey := make(map[npcDeckKey]EntityMBattleNpcDeck, len(npcDecks))
	for _, d := range npcDecks {
		npcDeckByKey[npcDeckKey{d.BattleNpcId, d.DeckType, d.BattleNpcDeckNumber}] = d
	}

	battleByIdMap := make(map[int32]EntityMBattle, len(battles))
	for _, b := range battles {
		battleByIdMap[b.BattleId] = b
	}

	type dropCatKey struct {
		BattleNpcId int64
		Uuid        string
	}
	dropCategoryByKey := make(map[dropCatKey]int32, len(npcDropCategories))
	for _, dc := range npcDropCategories {
		dropCategoryByKey[dropCatKey{dc.BattleNpcId, dc.BattleNpcDeckCharacterUuid}] = dc.BattleDropCategoryId
	}

	battleDropsByQuestId := make(map[int32][]BattleDropInfo)
	for questId := range questById {
		sids := sceneIdsByQuestId[questId]
		var drops []BattleDropInfo
		for _, sceneId := range sids {
			groupId, ok := battleGroupBySceneId[sceneId]
			if !ok {
				continue
			}
			for _, battleRow := range battleRowsByGroupId[groupId] {
				b, ok := battleByIdMap[battleRow.BattleId]
				if !ok {
					continue
				}
				dk := npcDeckKey{b.BattleNpcId, b.DeckType, b.BattleNpcDeckNumber}
				deck, ok := npcDeckByKey[dk]
				if !ok {
					continue
				}
				for _, uuid := range []string{deck.BattleNpcDeckCharacterUuid01, deck.BattleNpcDeckCharacterUuid02, deck.BattleNpcDeckCharacterUuid03} {
					if uuid == "" {
						continue
					}
					catId, ok := dropCategoryByKey[dropCatKey{b.BattleNpcId, uuid}]
					if !ok {
						continue
					}
					info := BattleDropInfo{QuestSceneId: sceneId, BattleDropCategoryId: catId}
					drops = append(drops, info)
				}
			}
		}
		if len(drops) > 0 {
			battleDropsByQuestId[questId] = drops
		}
	}
	bossCountByQuestId := buildBossCountByQuestId(scenes, sceneBattles, battleGroups, battles, npcDecks, npcCharacterTypes)

	rentalBattleGroups := make(map[int32]bool, len(rentalDecks))
	for _, rd := range rentalDecks {
		rentalBattleGroups[rd.BattleGroupId] = true
	}
	rentalQuestIds := make(map[int32]bool)
	for questId := range questById {
		for _, sceneId := range sceneIdsByQuestId[questId] {
			if groupId, ok := battleGroupBySceneId[sceneId]; ok && rentalBattleGroups[groupId] {
				rentalQuestIds[questId] = true
				break
			}
		}
	}

	return &QuestCatalog{
		SceneById:                          sceneById,
		MissionById:                        missionById,
		QuestById:                          questById,
		MissionIdsByQuestId:                missionIdsByQuestId,
		RouteIdByQuestId:                   routeIdByQuestId,
		MainQuestDifficultyTypeByQuestId:   mainQuestDifficultyTypeByQuestId,
		SceneIdsByQuestId:                  sceneIdsByQuestId,
		OrderedQuestIds:                    orderedQuestIds,
		FirstClearRewardsByGroupId:         firstClearRewardsByGroupId,
		FirstClearRewardSwitchesByQuestId:  firstClearRewardSwitchesByQuestId,
		MissionRewardsByMissionId:          missionRewardsByMissionId,
		MissionConditionValuesByGroupId:    missionConditionValuesByGroupId,
		SceneChoiceByKey:                   sceneChoiceByKey,
		WeaponIdsByReleaseConditionGroupId: weaponIdsByReleaseConditionGroupId,
		ReleaseConditionsByGroupId:         releaseConditionsByGroupId,
		SceneGrantsBySceneId:               sceneGrantsBySceneId,
		BattleDropRewardById:               battleDropRewardById,
		PickupRewardIdsByGroupId:           pickupRewardIdsByGroupId,
		PickupRewardIdsByGroupAndEffectId:  pickupRewardIdsByGroupAndEffectId,
		BattleDropEffectIdByRewardId:       battleDropEffectIdByRewardId,
		BattleDropsByQuestId:               battleDropsByQuestId,
		BossCountByQuestId:                 bossCountByQuestId,
		QuestBonusById:                     questBonusById,
		QuestBonusCharacterRowsByGroupId:   questBonusCharacterRowsByGroupId,
		QuestBonusWeaponRowsByGroupId:      questBonusWeaponRowsByGroupId,
		QuestBonusEffectsByGroupId:         questBonusEffectsByGroupId,
		QuestBonusDropByEffectId:           questBonusDropByEffectId,
		QuestBonusExpByEffectId:            questBonusExpByEffectId,
		QuestBonusTermsByGroupId:           questBonusTermsByGroupId,
		WeaponEvolutionByWeaponId:          weaponEvolutionByWeaponId,
		ReplayFlowRewardsByGroupId:         replayFlowRewardsByGroupId,
		RentalQuestIds:                     rentalQuestIds,
		TutorialUnlockConditions:           tutorialUnlockConds,
		ChapterLastSceneByQuestId:          chapterLastSceneByQuestId,
		SeasonIdByRouteId:                  seasonIdByRouteId,
		RoutesBySeason:                     routesBySeason,
		RouteCompletionQuestId:             routeCompletionQuestId,
		BattleOnlyTargetSceneByQuestId:     battleOnlyTargetSceneByQuestId,
		MainQuestChapterIdByQuestId:        mainQuestChapterIdByQuestId,
		MainQuestRouteIdByChapterId:        mainQuestRouteIdByChapterId,
		EventQuestTypeByChapterId:          eventQuestTypeByChapterId,
		EventChapterById:                   eventChapterById,
		EventQuestIdsByChapterId:           eventQuestIdsByChapterId,
		EventQuestIdsByChapterSortOrder:    eventQuestIdsByChapterSortOrder,
		EventQuestIdsByChapterDifficulty:   eventQuestIdsByChapterDifficulty,
		LimitContentQuestIds:               limitContentQuestIds,
		EventUnlockConditions:              eventUnlockConditions,
		EventCharacterIdsByChapterId:       eventCharacterIdsByChapterId,
		EventDailyGroups:                   eventDailyGroups,

		UserExpThresholds:       BuildExpThresholds(paramMapRows, 1),
		CharacterExpThresholds:  BuildExpThresholds(paramMapRows, 31),
		CostumeExpByRarity:      costumeExpByRarity,
		CostumeMaxLevelByRarity: costumeMaxLevelByRarity,
		MaxStaminaByLevel:       maxStaminaByLevel,

		CostumeById:         costumeById,
		CostumeEnhancedById: costumeEnhancedById,
		WeaponById:          weaponById,

		WeaponSkillSlots:   skillSlots,
		WeaponAbilitySlots: abilitySlots,

		PartsCatalog: partsCatalog,
	}, nil
}

func (q *QuestCatalog) BattleOnlyTargetSceneIdFor(questId int32) (int32, bool) {
	v, ok := q.BattleOnlyTargetSceneByQuestId[questId]
	return v, ok
}
