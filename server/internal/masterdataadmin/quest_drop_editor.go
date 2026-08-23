package masterdataadmin

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

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
	Value int32  `json:"value"`
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
	DropCount                int32                 `json:"dropCount"`
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
	PreviewRewardIDs         []int32            `json:"previewRewardIds"`
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

type mainStoryChapterSpec struct {
	chapterID           int32
	nameMissionTextID   int32
	mainQuestChapterIDs []int32
}

type mainQuestChapterDefinition struct {
	sequenceGroupID int32
}

var questDropTypes = []QuestDropType{
	{ID: "main", Value: 1, Label: "MAIN_QUEST"},
	{ID: "event-4", Value: 4, Label: "DAY_OF_THE_WEEK"},
	{ID: "event-5", Value: 5, Label: "GUERRILLA"},
	{ID: "event-6", Value: 6, Label: "CHARACTER"},
	{ID: "event-7", Value: 7, Label: "CHARACTER_QUEST"},
	{ID: "event-8", Value: 8, Label: "CAGE"},
	{ID: "event-10", Value: 10, Label: "TOWER"},
	{ID: "event-11", Value: 11, Label: "LIMIT_CONTENT"},
	{ID: "event-12", Value: 12, Label: "LABYRINTH"},
}

// MainQuestChapterId identifies internal route segments, not the 1-30 chapter
// number shown to players. These groups follow the main-story completion
// missions; prologue and joined hidden segments stay with their visible chapter.
var mainStoryChapterSpecs = []mainStoryChapterSpec{
	{1, 210001, []int32{1, 2}}, {2, 210002, []int32{3}}, {3, 210003, []int32{4}},
	{4, 210004, []int32{5}}, {5, 210005, []int32{6}}, {6, 210006, []int32{7}},
	{7, 210007, []int32{8, 9}}, {8, 210008, []int32{10}}, {9, 210017, []int32{11}},
	{10, 210019, []int32{12}}, {11, 210020, []int32{13}}, {12, 210021, []int32{14}},
	{13, 210030, []int32{15, 16}}, {14, 210031, []int32{23, 24}},
	{15, 210032, []int32{17}}, {16, 210033, []int32{25}},
	{17, 210034, []int32{18}}, {18, 210035, []int32{26}},
	{19, 210036, []int32{19, 20}}, {20, 210028, []int32{27, 28}},
	{21, 210038, []int32{21}}, {22, 210039, []int32{29}},
	{23, 210040, []int32{22}}, {24, 210041, []int32{30}},
	{25, 210060, []int32{31, 32}}, {26, 210061, []int32{33}},
	{27, 210062, []int32{34}}, {28, 210063, []int32{35}},
	{29, 210064, []int32{36}}, {30, 210065, []int32{37, 38}},
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
	for questID := range routesByQuest {
		sort.Slice(routesByQuest[questID], func(i, j int) bool {
			left, right := routesByQuest[questID][i], routesByQuest[questID][j]
			if left.PossessionType != right.PossessionType {
				return left.PossessionType < right.PossessionType
			}
			return left.PossessionID < right.PossessionID
		})
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

	groupByID := make(map[int32][]questdrop.Reward)
	groupRewardIndexes := make(map[int32]map[int32]int)
	type previewReward struct {
		sortOrder int64
		rowIndex  int
		rewardID  int32
	}
	previewByGroup := make(map[int32][]previewReward)
	for rowIndex, row := range readRows(file, questPickupRewardGroupTable) {
		groupID, groupOK := integerAt(row, 0)
		sortOrder, sortOK := integerAt(row, 1)
		rewardID, rewardOK := integerAt(row, 2)
		if !groupOK || !sortOK || !rewardOK || !includedGroups[int32(groupID)] {
			continue
		}
		group, reward := int32(groupID), int32(rewardID)
		previewByGroup[group] = append(previewByGroup[group], previewReward{
			sortOrder: sortOrder, rowIndex: rowIndex, rewardID: reward,
		})
		if groupRewardIndexes[group] == nil {
			groupRewardIndexes[group] = make(map[int32]int)
		}
		if index, exists := groupRewardIndexes[group][reward]; exists {
			groupByID[group][index].Weight++
		} else {
			groupRewardIndexes[group][reward] = len(groupByID[group])
			groupByID[group] = append(groupByID[group], questdrop.Reward{BattleDropRewardID: reward, Weight: 1})
		}
	}
	groupIDs := make([]int32, 0, len(groupByID))
	for groupID := range groupByID {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
	for _, groupID := range groupIDs {
		preview := previewByGroup[groupID]
		sort.SliceStable(preview, func(i, j int) bool {
			if preview[i].sortOrder != preview[j].sortOrder {
				return preview[i].sortOrder < preview[j].sortOrder
			}
			return preview[i].rowIndex < preview[j].rowIndex
		})
		previewRewardIDs := make([]int32, 0, len(preview))
		for _, reward := range preview {
			previewRewardIDs = append(previewRewardIDs, reward.rewardID)
		}
		result.Groups = append(result.Groups, QuestDropGroup{
			QuestPickupRewardGroupID: groupID,
			Rewards:                  groupByID[groupID],
			PreviewRewardIDs:         previewRewardIDs,
		})
	}

	rewardByID := make(map[int32]QuestDropReward)
	for _, row := range readRows(file, battleDropRewardTable) {
		reward, ok := questDropRewardAt(row)
		if !ok {
			continue
		}
		rewardByID[reward.BattleDropRewardID] = reward
	}
	for _, reward := range rewardByID {
		result.Rewards = append(result.Rewards, reward)
	}
	sort.Slice(result.Rewards, func(i, j int) bool {
		return result.Rewards[i].BattleDropRewardID < result.Rewards[j].BattleDropRewardID
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
	appendPlacements := func(typeID string, chapterID, sequenceGroupID int32, sequenceGroups, sequences [][]interface{}) {
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
	for _, chapter := range mainStoryQuestChapters(file) {
		chapters = append(chapters, QuestDropChapter{
			TypeID: "main", ChapterID: chapter.chapterID, SortOrder: chapter.chapterID,
			Names: mainStoryChapterNames(resolver, chapter.chapterID),
		})
		for _, definition := range chapter.definitions {
			appendPlacements("main", chapter.chapterID, definition.sequenceGroupID, mainGroups, mainSequences)
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
		if !chapterOK || !typeOK || !sortOK || !groupOK || eventType <= 3 || eventType == 9 || eventType > 12 {
			continue
		}
		var names map[string]string
		if nameOK && resolver != nil {
			names = resolver.byKey(fmt.Sprintf("quest.event.chapter_title.%d", nameTextID))
		}
		typeID := "event-" + strconv.FormatInt(eventType, 10)
		chapters = append(chapters, QuestDropChapter{
			TypeID: typeID, ChapterID: int32(chapterID), SortOrder: int32(sortOrder), Names: names,
		})
		appendPlacements(typeID, int32(chapterID), int32(groupID), eventGroups, eventSequences)
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
		return chapters[i].ChapterID < chapters[j].ChapterID
	})
	return placements, chapters
}

type mainStoryQuestChapter struct {
	chapterID   int32
	definitions []mainQuestChapterDefinition
}

func mainStoryQuestChapters(file *memorydb.File) []mainStoryQuestChapter {
	definitionsByID := make(map[int32]mainQuestChapterDefinition)
	for _, row := range readRows(file, "m_main_quest_chapter") {
		chapterID, chapterOK := integerAt(row, 0)
		groupID, groupOK := integerAt(row, 3)
		if !chapterOK || !groupOK {
			continue
		}
		id := int32(chapterID)
		definitionsByID[id] = mainQuestChapterDefinition{sequenceGroupID: int32(groupID)}
	}

	result := make([]mainStoryQuestChapter, 0, len(mainStoryChapterSpecs))
	for _, spec := range mainStoryChapterSpecs {
		chapter := mainStoryQuestChapter{chapterID: spec.chapterID}
		for _, chapterID := range spec.mainQuestChapterIDs {
			if definition, ok := definitionsByID[chapterID]; ok {
				chapter.definitions = append(chapter.definitions, definition)
			}
		}
		result = append(result, chapter)
	}
	return result
}

func mainStoryChapterNames(resolver *titleResolver, chapterID int32) map[string]string {
	if resolver == nil || chapterID < 1 || int(chapterID) > len(mainStoryChapterSpecs) {
		return nil
	}
	spec := mainStoryChapterSpecs[chapterID-1]
	names := resolver.byKey(fmt.Sprintf("mission.name.%d", spec.nameMissionTextID))
	if len(names) == 0 {
		return nil
	}
	for language, name := range names {
		switch language {
		case "en":
			name = strings.TrimSuffix(strings.TrimPrefix(name, "Clear "), " on Normal")
		case "ja":
			name = strings.TrimSuffix(name, "ノーマルをクリアする")
		case "ko":
			name = strings.TrimSuffix(name, "(Normal) 클리어하기")
			name = strings.TrimSuffix(name, " Normal 클리어하기")
		}
		names[language] = name
	}
	if name := names["ja"]; name != "" && chapterID >= 7 && chapterID <= 12 {
		chapterPrefixes := []string{"七章：", "八章：", "九章：", "十章：", "十一章：", "十二章："}
		nightPrefixes := []string{"第一夜：", "第二夜：", "第三夜：", "第四夜：", "第五夜：", "第六夜："}
		names["ja"] = strings.Replace(name, chapterPrefixes[chapterID-7], nightPrefixes[chapterID-7], 1)
	}
	return names
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
