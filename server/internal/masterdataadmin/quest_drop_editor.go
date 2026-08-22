package masterdataadmin

import (
	"fmt"
	"sort"
	"strconv"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/questdrop"
)

const (
	questTable                  = "m_quest"
	questPickupRewardGroupTable = "m_quest_pickup_reward_group"
	battleDropRewardTable       = "m_battle_drop_reward"
)

type QuestDropType struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type QuestDropChapter struct {
	TypeID    string            `json:"typeId"`
	ChapterID int32             `json:"chapterId"`
	SortOrder int32             `json:"sortOrder"`
	Names     map[string]string `json:"names,omitempty"`
}

type QuestDropPossession struct {
	PossessionType int32 `json:"possessionType"`
	PossessionID   int32 `json:"possessionId"`
}

type QuestDropQuest struct {
	QuestID                  int32                 `json:"questId"`
	TypeID                   string                `json:"typeId"`
	ChapterID                int32                 `json:"chapterId"`
	DifficultyType           int32                 `json:"difficultyType"`
	SortOrder                int32                 `json:"sortOrder"`
	QuestPickupRewardGroupID int32                 `json:"questPickupRewardGroupId"`
	RoutePossessions         []QuestDropPossession `json:"routePossessions,omitempty"`
}

type QuestDropReward struct {
	BattleDropRewardID int32 `json:"battleDropRewardId"`
	PossessionType     int32 `json:"possessionType"`
	PossessionID       int32 `json:"possessionId"`
	Count              int32 `json:"count"`
}

type QuestDropGroup struct {
	QuestPickupRewardGroupID int32              `json:"questPickupRewardGroupId"`
	Rewards                  []questdrop.Reward `json:"rewards"`
}

type QuestDropEditorCatalog struct {
	Types    []QuestDropType    `json:"types"`
	Chapters []QuestDropChapter `json:"chapters"`
	Quests   []QuestDropQuest   `json:"quests"`
	Groups   []QuestDropGroup   `json:"groups"`
	Rewards  []QuestDropReward  `json:"rewards"`
}

type questPlacement struct {
	questID        int32
	typeID         string
	chapterID      int32
	difficultyType int32
	sortOrder      int32
}

var questDropTypes = []QuestDropType{
	{ID: "main", Label: "主线副本"},
	{ID: "event-4", Label: "每日副本"},
	{ID: "event-5", Label: "游击副本"},
	{ID: "event-6", Label: "角色副本"},
	{ID: "event-7", Label: "角色剧情副本"},
	{ID: "event-8", Label: "牢笼副本"},
	{ID: "event-9", Label: "特殊副本"},
	{ID: "event-10", Label: "天顶之塔"},
	{ID: "event-11", Label: "限制副本"},
	{ID: "event-12", Label: "迷宫副本"},
}

func loadQuestDropEditor(file *memorydb.File, resolver *titleResolver) QuestDropEditorCatalog {
	placements, chapters := nonEventQuestPlacements(file, resolver)
	questRows := readRows(file, questTable)
	questGroupByID := make(map[int32]int32, len(questRows))
	for _, row := range questRows {
		questID, questOK := integerAt(row, 0)
		groupID, groupOK := integerAt(row, 8)
		if questOK && groupOK {
			questGroupByID[int32(questID)] = int32(groupID)
		}
	}

	routesByQuest := make(map[int32][]QuestDropPossession)
	for _, row := range readRows(file, "m_possession_acquisition_route") {
		possessionType, typeOK := integerAt(row, 0)
		possessionID, possessionOK := integerAt(row, 1)
		routeType, routeTypeOK := integerAt(row, 3)
		questID, questOK := integerAt(row, 4)
		if !typeOK || !possessionOK || !routeTypeOK || !questOK || routeType != 1 {
			continue
		}
		key := QuestDropPossession{PossessionType: int32(possessionType), PossessionID: int32(possessionID)}
		if !containsQuestDropPossession(routesByQuest[int32(questID)], key) {
			routesByQuest[int32(questID)] = append(routesByQuest[int32(questID)], key)
		}
	}

	result := QuestDropEditorCatalog{Types: append([]QuestDropType(nil), questDropTypes...), Chapters: chapters}
	includedGroups := make(map[int32]bool)
	for _, placement := range placements {
		groupID, exists := questGroupByID[placement.questID]
		if !exists {
			continue
		}
		result.Quests = append(result.Quests, QuestDropQuest{
			QuestID: placement.questID, TypeID: placement.typeID, ChapterID: placement.chapterID,
			DifficultyType: placement.difficultyType, SortOrder: placement.sortOrder,
			QuestPickupRewardGroupID: groupID, RoutePossessions: routesByQuest[placement.questID],
		})
		if groupID != 0 {
			includedGroups[groupID] = true
		}
	}

	selectedRewardIDs := make(map[int32]bool)
	groupByID := make(map[int32][]questdrop.Reward)
	groupRewardIndexes := make(map[int32]map[int32]int)
	for _, row := range readRows(file, questPickupRewardGroupTable) {
		groupID, groupOK := integerAt(row, 0)
		rewardID, rewardOK := integerAt(row, 2)
		if !groupOK || !rewardOK || !includedGroups[int32(groupID)] {
			continue
		}
		group, reward := int32(groupID), int32(rewardID)
		if groupRewardIndexes[group] == nil {
			groupRewardIndexes[group] = make(map[int32]int)
		}
		if index, exists := groupRewardIndexes[group][reward]; exists {
			groupByID[group][index].Weight++
		} else {
			groupRewardIndexes[group][reward] = len(groupByID[group])
			groupByID[group] = append(groupByID[group], questdrop.Reward{BattleDropRewardID: reward, Weight: 1})
		}
		selectedRewardIDs[reward] = true
	}
	groupIDs := make([]int32, 0, len(groupByID))
	for groupID := range groupByID {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
	for _, groupID := range groupIDs {
		result.Groups = append(result.Groups, QuestDropGroup{
			QuestPickupRewardGroupID: groupID,
			Rewards:                  groupByID[groupID],
		})
	}

	type rewardKey struct{ possessionType, possessionID, count int32 }
	canonicalByReward := make(map[rewardKey]QuestDropReward)
	rewardByID := make(map[int32]QuestDropReward)
	for _, row := range readRows(file, battleDropRewardTable) {
		reward, ok := questDropRewardAt(row)
		if !ok {
			continue
		}
		rewardByID[reward.BattleDropRewardID] = reward
		key := rewardKey{reward.PossessionType, reward.PossessionID, reward.Count}
		if previous, exists := canonicalByReward[key]; !exists || reward.BattleDropRewardID < previous.BattleDropRewardID {
			canonicalByReward[key] = reward
		}
	}
	includedRewardIDs := make(map[int32]bool, len(canonicalByReward)+len(selectedRewardIDs))
	for _, reward := range canonicalByReward {
		includedRewardIDs[reward.BattleDropRewardID] = true
	}
	for rewardID := range selectedRewardIDs {
		includedRewardIDs[rewardID] = true
	}
	for rewardID := range includedRewardIDs {
		if reward, exists := rewardByID[rewardID]; exists {
			result.Rewards = append(result.Rewards, reward)
		}
	}
	sort.Slice(result.Rewards, func(i, j int) bool {
		left, right := result.Rewards[i], result.Rewards[j]
		if left.PossessionType != right.PossessionType {
			return left.PossessionType < right.PossessionType
		}
		if left.PossessionID != right.PossessionID {
			return left.PossessionID < right.PossessionID
		}
		if left.Count != right.Count {
			return left.Count < right.Count
		}
		return left.BattleDropRewardID < right.BattleDropRewardID
	})
	return result
}

func containsQuestDropPossession(values []QuestDropPossession, target QuestDropPossession) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func questDropRewardAt(row []interface{}) (QuestDropReward, bool) {
	values := make([]int64, 4)
	for column := range values {
		value, ok := integerAt(row, column)
		if !ok {
			return QuestDropReward{}, false
		}
		values[column] = value
	}
	return QuestDropReward{
		BattleDropRewardID: int32(values[0]), PossessionType: int32(values[1]),
		PossessionID: int32(values[2]), Count: int32(values[3]),
	}, true
}

func nonEventQuestPlacements(file *memorydb.File, resolver *titleResolver) ([]questPlacement, []QuestDropChapter) {
	var placements []questPlacement
	var chapters []QuestDropChapter
	appendPlacements := func(typeID string, chapterID, chapterSort, sequenceGroupID int32, sequenceGroups, sequences [][]interface{}, names map[string]string) {
		chapters = append(chapters, QuestDropChapter{
			TypeID: typeID, ChapterID: chapterID, SortOrder: chapterSort, Names: names,
		})
		sequenceIDs := make(map[int32]int32)
		for _, row := range sequenceGroups {
			groupID, groupOK := integerAt(row, 0)
			difficulty, difficultyOK := integerAt(row, 1)
			sequenceID, sequenceOK := integerAt(row, 2)
			if groupOK && difficultyOK && sequenceOK && int32(groupID) == sequenceGroupID {
				sequenceIDs[int32(sequenceID)] = int32(difficulty)
			}
		}
		for _, row := range sequences {
			sequenceID, sequenceOK := integerAt(row, 0)
			sortOrder, sortOK := integerAt(row, 1)
			questID, questOK := integerAt(row, 2)
			difficulty, included := sequenceIDs[int32(sequenceID)]
			if !sequenceOK || !sortOK || !questOK || !included {
				continue
			}
			placements = append(placements, questPlacement{
				questID: int32(questID), typeID: typeID, chapterID: chapterID,
				difficultyType: difficulty, sortOrder: int32(sortOrder),
			})
		}
	}

	mainGroups := readRows(file, "m_main_quest_sequence_group")
	mainSequences := readRows(file, "m_main_quest_sequence")
	for _, row := range readRows(file, "m_main_quest_chapter") {
		chapterID, chapterOK := integerAt(row, 0)
		sortOrder, sortOK := integerAt(row, 2)
		groupID, groupOK := integerAt(row, 3)
		if chapterOK && sortOK && groupOK {
			appendPlacements("main", int32(chapterID), int32(sortOrder), int32(groupID), mainGroups, mainSequences, nil)
		}
	}

	eventGroups := readRows(file, "m_event_quest_sequence_group")
	eventSequences := readRows(file, "m_event_quest_sequence")
	for _, row := range readRows(file, "m_event_quest_chapter") {
		chapterID, chapterOK := integerAt(row, 0)
		eventType, typeOK := integerAt(row, 1)
		sortOrder, sortOK := integerAt(row, 2)
		nameTextID, nameOK := integerAt(row, 3)
		groupID, groupOK := integerAt(row, 7)
		if !chapterOK || !typeOK || !sortOK || !groupOK || eventType <= 3 || eventType > 12 {
			continue
		}
		var names map[string]string
		if nameOK && resolver != nil {
			names = resolver.byKey(fmt.Sprintf("quest.event.chapter_title.%d", nameTextID))
		}
		appendPlacements("event-"+strconv.FormatInt(eventType, 10), int32(chapterID), int32(sortOrder), int32(groupID), eventGroups, eventSequences, names)
	}

	sort.SliceStable(placements, func(i, j int) bool {
		if placements[i].typeID != placements[j].typeID {
			return questDropTypeOrder(placements[i].typeID) < questDropTypeOrder(placements[j].typeID)
		}
		if placements[i].chapterID != placements[j].chapterID {
			return placements[i].chapterID < placements[j].chapterID
		}
		if placements[i].difficultyType != placements[j].difficultyType {
			return placements[i].difficultyType < placements[j].difficultyType
		}
		if placements[i].sortOrder != placements[j].sortOrder {
			return placements[i].sortOrder < placements[j].sortOrder
		}
		return placements[i].questID < placements[j].questID
	})
	sort.SliceStable(chapters, func(i, j int) bool {
		if chapters[i].TypeID != chapters[j].TypeID {
			return questDropTypeOrder(chapters[i].TypeID) < questDropTypeOrder(chapters[j].TypeID)
		}
		if chapters[i].SortOrder != chapters[j].SortOrder {
			return chapters[i].SortOrder < chapters[j].SortOrder
		}
		return chapters[i].ChapterID < chapters[j].ChapterID
	})
	return placements, chapters
}

func questDropTypeOrder(typeID string) int {
	for index, definition := range questDropTypes {
		if definition.ID == typeID {
			return index
		}
	}
	return len(questDropTypes)
}

func ValidateQuestDropConfigScope(editor QuestDropEditorCatalog, config *questdrop.Config) error {
	if config == nil {
		return fmt.Errorf("quest drop config is nil")
	}
	configurable := make(map[int32]bool, len(editor.Quests))
	for _, quest := range editor.Quests {
		configurable[quest.QuestID] = true
	}
	for questID := range config.Quests {
		if !configurable[questID] {
			return fmt.Errorf("QuestId %d is not a configurable non-event quest", questID)
		}
	}
	return nil
}
