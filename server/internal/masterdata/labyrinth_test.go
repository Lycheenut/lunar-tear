package masterdata

import "testing"

func TestLabyrinthSeasonUsesEndedSeasonRewardGroup(t *testing.T) {
	catalog := &LabyrinthCatalog{
		SeasonsByChapter: map[int32]map[int32]EntityMEventQuestLabyrinthSeason{
			10: {
				1: {EventQuestChapterId: 10, SeasonNumber: 1, EndDatetime: 100, SeasonRewardGroupId: 11},
				2: {EventQuestChapterId: 10, SeasonNumber: 2, EndDatetime: 200, SeasonRewardGroupId: 12},
				3: {EventQuestChapterId: 10, SeasonNumber: 3, EndDatetime: 300, SeasonRewardGroupId: 13},
			},
		},
		SeasonMilestonesByRewardGroup: map[int32][]LabyrinthSeasonMilestone{
			12: {{HeadQuestId: 120}},
			13: {{HeadQuestId: 130}},
		},
	}
	season, ok := catalog.LatestEndedSeason(10, 250)
	if !ok || season.SeasonNumber != 2 {
		t.Fatalf("ended season = %+v, %v", season, ok)
	}
	if milestones := catalog.SeasonMilestonesFor(season); len(milestones) != 1 || milestones[0].HeadQuestId != 120 {
		t.Fatalf("season milestones = %+v", milestones)
	}
}

func TestLabyrinthStageQuestIdsUsesConfiguredSortRange(t *testing.T) {
	catalog := &LabyrinthCatalog{StagesByKey: map[labyrinthStageKey]EntityMEventQuestLabyrinthStage{
		{ChapterId: 10, StageOrder: 2}: {EventQuestChapterId: 10, StageOrder: 2, StartSequenceSortOrder: 2, EndSequenceSortOrder: 3},
	}}
	quests := &QuestCatalog{EventQuestIdsByChapterSortOrder: map[int32]map[int32][]int32{
		10: {1: {100}, 2: {200}, 3: {300}, 4: {400}},
	}}
	questIds, ok := catalog.StageQuestIds(quests, 10, 2)
	if !ok || len(questIds) != 2 || questIds[0] != 200 || questIds[1] != 300 {
		t.Fatalf("stage quests = %v, ok=%v", questIds, ok)
	}
	delete(quests.EventQuestIdsByChapterSortOrder[10], 3)
	if _, ok := catalog.StageQuestIds(quests, 10, 2); ok {
		t.Fatal("stage mapping with a missing sort order was accepted")
	}
}
