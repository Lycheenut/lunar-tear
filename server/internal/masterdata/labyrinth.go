package masterdata

import (
	"log"
	"sort"

	"lunar-tear/server/internal/utils"
)

type LabyrinthChapter struct {
	EventQuestChapterId int32
	LatestSeasonNumber  int32
	StageOrders         []int32
}

type LabyrinthStageTier struct {
	QuestMissionClearCount int32
	Rewards                []RewardItem
}

type LabyrinthSeasonMilestone struct {
	HeadQuestId    int32
	HeadStageOrder int32
	Rewards        []RewardItem
}

type labyrinthStageKey struct {
	ChapterId  int32
	StageOrder int32
}

type LabyrinthCatalog struct {
	ChaptersByOrder               []LabyrinthChapter
	StagesByKey                   map[labyrinthStageKey]EntityMEventQuestLabyrinthStage
	ClearRewardsByStage           map[labyrinthStageKey][]RewardItem
	AccumTiersByStage             map[labyrinthStageKey][]LabyrinthStageTier
	SeasonMilestonesByRewardGroup map[int32][]LabyrinthSeasonMilestone
	SeasonsByChapter              map[int32]map[int32]EntityMEventQuestLabyrinthSeason
}

func (c *LabyrinthCatalog) HasStage(chapterId, stageOrder int32) bool {
	_, ok := c.StagesByKey[labyrinthStageKey{chapterId, stageOrder}]
	return ok
}

func (c *LabyrinthCatalog) StageQuestIds(quests *QuestCatalog, chapterId, stageOrder int32) ([]int32, bool) {
	stage, ok := c.StagesByKey[labyrinthStageKey{chapterId, stageOrder}]
	if !ok || stage.StartSequenceSortOrder <= 0 || stage.EndSequenceSortOrder < stage.StartSequenceSortOrder {
		return nil, false
	}
	bySortOrder := quests.EventQuestIdsByChapterSortOrder[chapterId]
	var questIds []int32
	for sortOrder := stage.StartSequenceSortOrder; sortOrder <= stage.EndSequenceSortOrder; sortOrder++ {
		ids := bySortOrder[sortOrder]
		if len(ids) == 0 {
			return nil, false
		}
		questIds = append(questIds, ids...)
	}
	return questIds, len(questIds) > 0
}

func (c *LabyrinthCatalog) LatestStartedSeason(chapterId int32, nowMillis int64) (EntityMEventQuestLabyrinthSeason, bool) {
	var latest EntityMEventQuestLabyrinthSeason
	for _, season := range c.SeasonsByChapter[chapterId] {
		if season.StartDatetime <= nowMillis && season.SeasonNumber > latest.SeasonNumber {
			latest = season
		}
	}
	return latest, latest.SeasonNumber != 0
}

func (c *LabyrinthCatalog) Season(chapterId, seasonNumber int32) (EntityMEventQuestLabyrinthSeason, bool) {
	season, ok := c.SeasonsByChapter[chapterId][seasonNumber]
	return season, ok
}

func (c *LabyrinthCatalog) StageClearReward(chapterId, stageOrder int32) []RewardItem {
	return c.ClearRewardsByStage[labyrinthStageKey{chapterId, stageOrder}]
}

func (c *LabyrinthCatalog) CollectAccumulationRewards(chapterId, stageOrder, oldCount, targetCount int32) ([]RewardItem, int32) {
	var items []RewardItem
	highest := int32(0)
	for _, t := range c.AccumTiersByStage[labyrinthStageKey{chapterId, stageOrder}] {
		if t.QuestMissionClearCount > oldCount && t.QuestMissionClearCount <= targetCount {
			items = append(items, t.Rewards...)
			if t.QuestMissionClearCount > highest {
				highest = t.QuestMissionClearCount
			}
		}
	}
	return items, highest
}

func (c *LabyrinthCatalog) SeasonMilestonesFor(season EntityMEventQuestLabyrinthSeason) []LabyrinthSeasonMilestone {
	return c.SeasonMilestonesByRewardGroup[season.SeasonRewardGroupId]
}

func LoadLabyrinthCatalog() *LabyrinthCatalog {
	seasonRows, err := utils.ReadTable[EntityMEventQuestLabyrinthSeason]("m_event_quest_labyrinth_season")
	if err != nil {
		log.Printf("[labyrinth] m_event_quest_labyrinth_season unavailable, labyrinth disabled: %v", err)
		return &LabyrinthCatalog{}
	}
	stageRows, err := utils.ReadTable[EntityMEventQuestLabyrinthStage]("m_event_quest_labyrinth_stage")
	if err != nil {
		log.Printf("[labyrinth] m_event_quest_labyrinth_stage unavailable, labyrinth disabled: %v", err)
		return &LabyrinthCatalog{}
	}

	// chapterId -> highest SeasonNumber
	latestSeason := make(map[int32]int32)
	for _, r := range seasonRows {
		if r.SeasonNumber > latestSeason[r.EventQuestChapterId] {
			latestSeason[r.EventQuestChapterId] = r.SeasonNumber
		}
	}
	// chapterId -> stage orders
	stagesByChapter := make(map[int32][]int32)
	stagesByKey := make(map[labyrinthStageKey]EntityMEventQuestLabyrinthStage, len(stageRows))
	for _, r := range stageRows {
		stagesByChapter[r.EventQuestChapterId] = append(stagesByChapter[r.EventQuestChapterId], r.StageOrder)
		stagesByKey[labyrinthStageKey{r.EventQuestChapterId, r.StageOrder}] = r
	}

	chapters := make([]LabyrinthChapter, 0, len(latestSeason))
	for chapterId, season := range latestSeason {
		stages := stagesByChapter[chapterId]
		sort.Slice(stages, func(i, j int) bool { return stages[i] < stages[j] })
		chapters = append(chapters, LabyrinthChapter{
			EventQuestChapterId: chapterId,
			LatestSeasonNumber:  season,
			StageOrders:         stages,
		})
	}
	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].EventQuestChapterId < chapters[j].EventQuestChapterId
	})

	clearRewards, accumTiers, seasonMilestones := loadLabyrinthRewards(stageRows)
	seasonsByChapter := make(map[int32]map[int32]EntityMEventQuestLabyrinthSeason)
	for _, season := range seasonRows {
		if seasonsByChapter[season.EventQuestChapterId] == nil {
			seasonsByChapter[season.EventQuestChapterId] = make(map[int32]EntityMEventQuestLabyrinthSeason)
		}
		seasonsByChapter[season.EventQuestChapterId][season.SeasonNumber] = season
	}

	log.Printf("labyrinth catalog loaded: %d chapters, %d stages with clear rewards, %d with accumulation rewards, %d season reward groups",
		len(chapters), len(clearRewards), len(accumTiers), len(seasonMilestones))
	return &LabyrinthCatalog{
		ChaptersByOrder:               chapters,
		StagesByKey:                   stagesByKey,
		ClearRewardsByStage:           clearRewards,
		AccumTiersByStage:             accumTiers,
		SeasonMilestonesByRewardGroup: seasonMilestones,
		SeasonsByChapter:              seasonsByChapter,
	}
}

func loadLabyrinthRewards(stageRows []EntityMEventQuestLabyrinthStage) (
	clearRewards map[labyrinthStageKey][]RewardItem,
	accumTiers map[labyrinthStageKey][]LabyrinthStageTier,
	seasonMilestones map[int32][]LabyrinthSeasonMilestone,
) {
	rewardGroupRows, err := utils.ReadTable[EntityMEventQuestLabyrinthRewardGroup]("m_event_quest_labyrinth_reward_group")
	if err != nil {
		log.Printf("[labyrinth] m_event_quest_labyrinth_reward_group unavailable, rewards disabled: %v", err)
		return nil, nil, nil
	}

	// reward group id -> reward items
	itemsByRewardGroup := make(map[int32][]RewardItem)
	for _, r := range rewardGroupRows {
		itemsByRewardGroup[r.EventQuestLabyrinthRewardGroupId] = append(itemsByRewardGroup[r.EventQuestLabyrinthRewardGroupId], RewardItem{
			PossessionType: r.PossessionType,
			PossessionId:   r.PossessionId,
			Count:          r.Count,
		})
	}

	// per-stage one-time clear reward
	clearRewards = make(map[labyrinthStageKey][]RewardItem)
	for _, r := range stageRows {
		if r.StageClearRewardGroupId == 0 {
			continue
		}
		if items := itemsByRewardGroup[r.StageClearRewardGroupId]; len(items) > 0 {
			clearRewards[labyrinthStageKey{r.EventQuestChapterId, r.StageOrder}] = items
		}
	}

	if accumGroupRows, err := utils.ReadTable[EntityMEventQuestLabyrinthStageAccumulationRewardGroup]("m_event_quest_labyrinth_stage_accumulation_reward_group"); err != nil {
		log.Printf("[labyrinth] m_event_quest_labyrinth_stage_accumulation_reward_group unavailable, accumulation rewards disabled: %v", err)
	} else {
		// accumulation group id -> tiers (threshold + resolved reward items)
		tiersByGroup := make(map[int32][]LabyrinthStageTier)
		for _, r := range accumGroupRows {
			tiersByGroup[r.EventQuestLabyrinthStageAccumulationRewardGroupId] = append(tiersByGroup[r.EventQuestLabyrinthStageAccumulationRewardGroupId], LabyrinthStageTier{
				QuestMissionClearCount: r.QuestMissionClearCount,
				Rewards:                itemsByRewardGroup[r.EventQuestLabyrinthRewardGroupId],
			})
		}
		accumTiers = make(map[labyrinthStageKey][]LabyrinthStageTier)
		for _, r := range stageRows {
			if r.StageAccumulationRewardGroupId == 0 {
				continue
			}
			tiers := tiersByGroup[r.StageAccumulationRewardGroupId]
			sort.Slice(tiers, func(i, j int) bool {
				return tiers[i].QuestMissionClearCount < tiers[j].QuestMissionClearCount
			})
			accumTiers[labyrinthStageKey{r.EventQuestChapterId, r.StageOrder}] = tiers
		}
	}

	// per-chapter season-reward milestones
	if seasonRewardRows, err := utils.ReadTable[EntityMEventQuestLabyrinthSeasonRewardGroup]("m_event_quest_labyrinth_season_reward_group"); err != nil {
		log.Printf("[labyrinth] m_event_quest_labyrinth_season_reward_group unavailable, season rewards disabled: %v", err)
	} else {
		seasonMilestones = buildLabyrinthSeasonMilestones(seasonRewardRows, itemsByRewardGroup)
	}

	return clearRewards, accumTiers, seasonMilestones
}

func buildLabyrinthSeasonMilestones(
	seasonRewardRows []EntityMEventQuestLabyrinthSeasonRewardGroup,
	itemsByRewardGroup map[int32][]RewardItem,
) map[int32][]LabyrinthSeasonMilestone {
	// SeasonRewardGroupId -> its rows, in table order
	rowsByGroup := make(map[int32][]EntityMEventQuestLabyrinthSeasonRewardGroup)
	for _, r := range seasonRewardRows {
		rowsByGroup[r.EventQuestLabyrinthSeasonRewardGroupId] = append(rowsByGroup[r.EventQuestLabyrinthSeasonRewardGroupId], r)
	}

	milestones := make(map[int32][]LabyrinthSeasonMilestone, len(rowsByGroup))
	for seasonGroupId, rows := range rowsByGroup {
		if len(rows) == 0 {
			continue
		}
		// rank distinct reward-group ids ascending -> 1-based head stage order
		stageByRewardGroup := make(map[int32]int32)
		var distinct []int32
		for _, r := range rows {
			if _, seen := stageByRewardGroup[r.EventQuestLabyrinthRewardGroupId]; !seen {
				stageByRewardGroup[r.EventQuestLabyrinthRewardGroupId] = 0
				distinct = append(distinct, r.EventQuestLabyrinthRewardGroupId)
			}
		}
		sort.Slice(distinct, func(i, j int) bool { return distinct[i] < distinct[j] })
		for i, gid := range distinct {
			stageByRewardGroup[gid] = int32(i + 1)
		}

		list := make([]LabyrinthSeasonMilestone, 0, len(rows))
		for _, r := range rows {
			list = append(list, LabyrinthSeasonMilestone{
				HeadQuestId:    r.HeadQuestId,
				HeadStageOrder: stageByRewardGroup[r.EventQuestLabyrinthRewardGroupId],
				Rewards:        itemsByRewardGroup[r.EventQuestLabyrinthRewardGroupId],
			})
		}
		milestones[seasonGroupId] = list
	}
	return milestones
}
