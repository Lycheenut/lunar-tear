package service

import (
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestSyncMissionProgressUsesMeasuredValues(t *testing.T) {
	catalog := &masterdata.MissionCatalog{
		MissionById:                map[int32]masterdata.EntityMMission{1: {MissionId: 1, MissionClearConditionType: 37, ClearConditionValue: 100}},
		MeasurableMissionIdsByType: map[int32][]int32{37: {1}},
		TermById:                   map[int32]masterdata.EntityMMissionTerm{},
		UnlockById:                 map[int32]masterdata.EntityMMissionUnlockCondition{},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	if err := syncMissionProgress(catalog, user, &pb.UpdateMissionProgressRequest{CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 99}}, 2); err != nil {
		t.Fatal(err)
	}
	if user.Missions[1].MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatal("mission cleared below target")
	}
	if err := syncMissionProgress(catalog, user, &pb.UpdateMissionProgressRequest{CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 100}}, 3); err != nil {
		t.Fatal(err)
	}
	if user.Missions[1].MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatal("mission did not clear at target")
	}
}

func TestSyncMissionProgressOnlyCreatesReportedMeasurableMissions(t *testing.T) {
	catalog := &masterdata.MissionCatalog{
		MissionById: map[int32]masterdata.EntityMMission{
			1: {MissionId: 1, MissionClearConditionType: 37, ClearConditionValue: 10},
			2: {MissionId: 2, MissionClearConditionType: 39, ClearConditionValue: 10},
			3: {MissionId: 3, MissionClearConditionType: 99, ClearConditionValue: 10},
		},
		MeasurableMissionIdsByType: map[int32][]int32{37: {1}, 39: {2}},
		TermById:                   map[int32]masterdata.EntityMMissionTerm{},
		UnlockById:                 map[int32]masterdata.EntityMMissionUnlockCondition{},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	if err := syncMissionProgress(catalog, user, &pb.UpdateMissionProgressRequest{CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 5}}, 2); err != nil {
		t.Fatal(err)
	}
	if _, ok := user.Missions[1]; !ok {
		t.Fatal("reported cage mission was not initialized")
	}
	if _, ok := user.Missions[2]; ok {
		t.Fatal("unreported picture-book mission was initialized")
	}
	if _, ok := user.Missions[3]; ok {
		t.Fatal("unsupported mission was initialized")
	}
}

func TestSyncMissionProgressRejectsInvalidRhythmMetricWithoutMutation(t *testing.T) {
	catalog := &masterdata.MissionCatalog{
		MissionById:                map[int32]masterdata.EntityMMission{1: {MissionId: 1, MissionClearConditionType: 36, ClearConditionValue: 1}},
		MeasurableMissionIdsByType: map[int32][]int32{36: {1}},
		TermById:                   map[int32]masterdata.EntityMMissionTerm{},
		UnlockById:                 map[int32]masterdata.EntityMMissionUnlockCondition{},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	err := syncMissionProgress(catalog, user, &pb.UpdateMissionProgressRequest{PictureBookMeasurableValues: &pb.PictureBookMeasurableValues{RhythmInteractionMeasurableValues: &pb.RhythmInteractionMeasurableValues{TapCount: 1}}}, 2)
	if err == nil {
		t.Fatal("rhythm metric without live type was accepted")
	}
	if _, ok := user.Missions[1]; ok {
		t.Fatal("invalid rhythm metric mutated mission state")
	}
}

func TestSyncMissionProgressIgnoresEmptyRhythmMetric(t *testing.T) {
	catalog := &masterdata.MissionCatalog{
		MissionById:                map[int32]masterdata.EntityMMission{1: {MissionId: 1, MissionClearConditionType: 36, ClearConditionValue: 1}},
		MeasurableMissionIdsByType: map[int32][]int32{36: {1}},
		TermById:                   map[int32]masterdata.EntityMMissionTerm{},
		UnlockById:                 map[int32]masterdata.EntityMMissionUnlockCondition{},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	err := syncMissionProgress(catalog, user, &pb.UpdateMissionProgressRequest{PictureBookMeasurableValues: &pb.PictureBookMeasurableValues{RhythmInteractionMeasurableValues: &pb.RhythmInteractionMeasurableValues{}}}, 2)
	if err != nil {
		t.Fatalf("empty rhythm metric was rejected: %v", err)
	}
	if _, ok := user.Missions[1]; ok {
		t.Fatal("empty rhythm metric mutated mission state")
	}
}

func TestClaimMissionPassRewardsIsIdempotent(t *testing.T) {
	cat := &runtime.Catalogs{Mission: &masterdata.MissionCatalog{PassById: map[int32]masterdata.MissionPassCatalog{7: {Definition: masterdata.EntityMMissionPass{MissionPassId: 7, StartDatetime: 1, EndDatetime: 100}, Levels: []masterdata.EntityMMissionPassLevelGroup{{Level: 1, NecessaryPoint: 10}}, Rewards: []masterdata.EntityMMissionPassRewardGroup{{Level: 1, PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 20, Count: 2}}}}}, QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.MissionPassPoints[7] = store.MissionPassPointState{MissionPassId: 7, Point: 10}
	first, err := claimMissionPassRewards(cat, user, 7, 50, false)
	if err != nil || len(first) != 1 || user.Materials[20] != 2 {
		t.Fatalf("first claim failed: rewards=%v materials=%d err=%v", first, user.Materials[20], err)
	}
	second, err := claimMissionPassRewards(cat, user, 7, 60, false)
	if err != nil || len(second) != 0 || user.Materials[20] != 2 {
		t.Fatalf("duplicate claim changed rewards: rewards=%v materials=%d err=%v", second, user.Materials[20], err)
	}
}

func TestOpenEndedMissionPassRemainsActiveAndNeverCountsAsEnded(t *testing.T) {
	pass := masterdata.EntityMMissionPass{StartDatetime: 10}
	if !missionPassActive(pass, 20) {
		t.Fatal("open-ended mission pass was treated as inactive")
	}
	if missionPassEnded(pass, 20) {
		t.Fatal("open-ended mission pass was treated as ended")
	}
}
