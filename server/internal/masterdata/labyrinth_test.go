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
