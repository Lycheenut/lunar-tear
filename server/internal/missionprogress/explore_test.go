package missionprogress

import (
	"fmt"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestExploreHighScoreMatchesMissionOptions(t *testing.T) {
	tests := []struct {
		name   string
		option int32
		ids    []int32
	}{
		{"shooting unspecified difficulty", 3, []int32{1, 11}},
		{"flying mama unspecified difficulty", 7, []int32{2, 12}},
		{"shooting hard", 26, []int32{11}},
		{"flying mama hard", 27, []int32{12}},
		{"shooting any difficulty", 28, []int32{1, 11}},
		{"flying mama any difficulty", 29, []int32{2, 12}},
		{"any explore", 0, []int32{1, 2, 11, 12}},
		{"direct explore ID", 11, []int32{11}},
		{"unknown option", 999, nil},
	}
	for _, test := range tests {
		for _, exploreId := range []int32{1, 2, 11, 12} {
			for _, source := range []string{"saved score", "score event"} {
				t.Run(fmt.Sprintf("%s/%d/%s", test.name, exploreId, source), func(t *testing.T) {
					mission := masterdata.EntityMMission{
						MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeExploreHighScore),
						MissionClearConditionOptionGroupId: test.option, ClearConditionValue: 100_000,
					}
					catalogs := testCatalog(mission)
					user := &store.UserState{}
					user.EnsureMaps()
					Sync(catalogs, user, 1)
					if source == "saved score" {
						user.ExploreScores[exploreId] = store.ExploreScoreState{ExploreId: exploreId, MaxScore: 100_000}
						Sync(catalogs, user, 2)
					} else {
						Apply(catalogs, nil, user, []store.MissionEvent{{
							ConditionType: int32(model.MissionClearConditionTypeExploreHighScore),
							TargetId:      exploreId, IsValue: true, Value: 100_000,
						}}, 2)
					}
					wantProgress, wantStatus := int32(0), int32(model.MissionProgressStatusTypeInProgress)
					if containsTarget(test.ids, exploreId) {
						wantProgress, wantStatus = 100_000, int32(model.MissionProgressStatusTypeClear)
					}
					if state := user.Missions[1]; state.ProgressValue != wantProgress || state.MissionProgressStatusType != wantStatus {
						t.Fatalf("mission = %+v, want progress %d status %d", state, wantProgress, wantStatus)
					}
				})
			}
		}
	}
}

func TestExploreHighScoreDoesNotSumDifficulties(t *testing.T) {
	catalogs := testCatalog(masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeExploreHighScore),
		MissionClearConditionOptionGroupId: 28, ClearConditionValue: 100_000,
	})
	user := &store.UserState{ExploreScores: map[int32]store.ExploreScoreState{
		1: {ExploreId: 1, MaxScore: 60_000}, 11: {ExploreId: 11, MaxScore: 70_000},
		2: {ExploreId: 2, MaxScore: 120_000},
	}}
	Sync(catalogs, user, 1)
	if state := user.Missions[1]; state.ProgressValue != 70_000 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("mission should use the highest matching score: %+v", state)
	}
}
