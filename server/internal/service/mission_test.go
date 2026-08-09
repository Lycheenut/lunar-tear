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
		MissionById:                map[int32]masterdata.EntityMMission{1: {MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeTowerWalkedDistance), ClearConditionValue: 100}},
		MeasurableMissionIdsByType: map[int32][]int32{int32(model.MissionClearConditionTypeTowerWalkedDistance): {1}},
		TermById:                   map[int32]masterdata.EntityMMissionTerm{},
		UnlockById:                 map[int32]masterdata.EntityMMissionUnlockCondition{},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	if err := syncMissionProgress(&runtime.Catalogs{Mission: catalog}, user, &pb.UpdateMissionProgressRequest{CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 99}}, 2); err != nil {
		t.Fatal(err)
	}
	if user.Missions[1].MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatal("mission cleared below target")
	}
	if err := syncMissionProgress(&runtime.Catalogs{Mission: catalog}, user, &pb.UpdateMissionProgressRequest{CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 100}}, 3); err != nil {
		t.Fatal(err)
	}
	if user.Missions[1].MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatal("mission did not clear at target")
	}
}

func TestSyncMissionProgressCreatesAllUnlockedKnownMissions(t *testing.T) {
	catalog := &masterdata.MissionCatalog{
		MissionById: map[int32]masterdata.EntityMMission{
			1: {MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeTowerWalkedDistance), ClearConditionValue: 10},
			2: {MissionId: 2, MissionClearConditionType: int32(model.MissionClearConditionTypeDefeatWizardCount), ClearConditionValue: 10},
			3: {MissionId: 3, MissionClearConditionType: 99, ClearConditionValue: 10},
		},
		MeasurableMissionIdsByType: map[int32][]int32{
			int32(model.MissionClearConditionTypeTowerWalkedDistance): {1},
			int32(model.MissionClearConditionTypeDefeatWizardCount):   {2},
		},
		TermById:   map[int32]masterdata.EntityMMissionTerm{},
		UnlockById: map[int32]masterdata.EntityMMissionUnlockCondition{},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	if err := syncMissionProgress(&runtime.Catalogs{Mission: catalog}, user, &pb.UpdateMissionProgressRequest{CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 5}}, 2); err != nil {
		t.Fatal(err)
	}
	if _, ok := user.Missions[1]; !ok {
		t.Fatal("reported cage mission was not initialized")
	}
	if state, ok := user.Missions[2]; !ok || state.ProgressValue != 0 {
		t.Fatal("unreported unlocked mission was not initialized with zero progress")
	}
	if _, ok := user.Missions[3]; ok {
		t.Fatal("unsupported mission was initialized")
	}
}

func TestSyncMissionProgressRejectsInvalidRhythmMetricWithoutMutation(t *testing.T) {
	catalog := &masterdata.MissionCatalog{
		MissionById:                map[int32]masterdata.EntityMMission{1: {MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeRhythmInteractionTapCount), ClearConditionValue: 1}},
		MeasurableMissionIdsByType: map[int32][]int32{int32(model.MissionClearConditionTypeRhythmInteractionTapCount): {1}},
		TermById:                   map[int32]masterdata.EntityMMissionTerm{},
		UnlockById:                 map[int32]masterdata.EntityMMissionUnlockCondition{},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	err := syncMissionProgress(&runtime.Catalogs{Mission: catalog}, user, &pb.UpdateMissionProgressRequest{PictureBookMeasurableValues: &pb.PictureBookMeasurableValues{RhythmInteractionMeasurableValues: &pb.RhythmInteractionMeasurableValues{TapCount: 1}}}, 2)
	if err == nil {
		t.Fatal("rhythm metric without live type was accepted")
	}
	if _, ok := user.Missions[1]; ok {
		t.Fatal("invalid rhythm metric mutated mission state")
	}
}

func TestSyncMissionProgressIgnoresEmptyRhythmMetricValue(t *testing.T) {
	catalog := &masterdata.MissionCatalog{
		MissionById:                map[int32]masterdata.EntityMMission{1: {MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeRhythmInteractionTapCount), ClearConditionValue: 1}},
		MeasurableMissionIdsByType: map[int32][]int32{int32(model.MissionClearConditionTypeRhythmInteractionTapCount): {1}},
		TermById:                   map[int32]masterdata.EntityMMissionTerm{},
		UnlockById:                 map[int32]masterdata.EntityMMissionUnlockCondition{},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	err := syncMissionProgress(&runtime.Catalogs{Mission: catalog}, user, &pb.UpdateMissionProgressRequest{PictureBookMeasurableValues: &pb.PictureBookMeasurableValues{RhythmInteractionMeasurableValues: &pb.RhythmInteractionMeasurableValues{}}}, 2)
	if err != nil {
		t.Fatalf("empty rhythm metric was rejected: %v", err)
	}
	if state, ok := user.Missions[1]; !ok || state.ProgressValue != 0 {
		t.Fatal("empty rhythm metric changed mission progress")
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

func TestClaimMissionRewardsByCategoryClaimsOnlyClearMissionsInCategory(t *testing.T) {
	cat := &runtime.Catalogs{
		Mission: &masterdata.MissionCatalog{
			MissionById: map[int32]masterdata.EntityMMission{
				1: {MissionId: 1, MissionGroupId: 11, MissionRewardId: 101},
				2: {MissionId: 2, MissionGroupId: 22, MissionRewardId: 102},
				3: {MissionId: 3, MissionGroupId: 11, MissionRewardId: 103},
			},
			GroupById: map[int32]masterdata.EntityMMissionGroup{
				11: {MissionGroupId: 11, MissionCategoryType: 1},
				22: {MissionGroupId: 22, MissionCategoryType: 2},
			},
			RewardsById: map[int32][]masterdata.EntityMMissionReward{
				101: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 31, Count: 1}},
				102: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 32, Count: 1}},
				103: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 33, Count: 1}},
			},
			TermById: map[int32]masterdata.EntityMMissionTerm{},
		},
		QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}},
	}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Missions[1] = store.UserMissionState{MissionId: 1, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
	user.Missions[2] = store.UserMissionState{MissionId: 2, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
	user.Missions[3] = store.UserMissionState{MissionId: 3, MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress)}

	received, expired := claimMissionRewardsByCategory(cat, user, 1, 100)
	if len(received) != 1 || len(expired) != 0 || user.Materials[31] != 1 {
		t.Fatalf("unexpected category rewards: received=%v expired=%v material31=%d", received, expired, user.Materials[31])
	}
	if user.Materials[32] != 0 || user.Materials[33] != 0 {
		t.Fatal("category claim granted a reward outside its clear missions")
	}
	if user.Missions[1].MissionProgressStatusType != int32(model.MissionProgressStatusTypeRewardReceived) || user.Missions[2].MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatal("category claim changed the wrong mission state")
	}
}

func TestClaimMissionRewardsSkipsNonClaimableCandidates(t *testing.T) {
	cat := &runtime.Catalogs{
		Mission: &masterdata.MissionCatalog{
			MissionById: map[int32]masterdata.EntityMMission{
				1:      {MissionId: 1, MissionRewardId: 101},
				200001: {MissionId: 200001, MissionRewardId: 102},
				3:      {MissionId: 3, MissionRewardId: 103},
			},
			RewardsById: map[int32][]masterdata.EntityMMissionReward{
				101: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 31, Count: 1}},
				102: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 32, Count: 1}},
				103: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 33, Count: 1}},
			},
			TermById: map[int32]masterdata.EntityMMissionTerm{},
		},
		QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}},
	}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Missions[1] = store.UserMissionState{MissionId: 1, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
	user.Missions[200001] = store.UserMissionState{MissionId: 200001, MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress)}
	user.Missions[3] = store.UserMissionState{MissionId: 3, MissionProgressStatusType: int32(model.MissionProgressStatusTypeRewardReceived)}

	received, expired := claimMissionRewards(cat, user, []int32{1, 200001, 3}, 100)
	if len(received) != 1 || len(expired) != 0 || user.Materials[31] != 1 {
		t.Fatalf("unexpected candidate rewards: received=%v expired=%v material31=%d", received, expired, user.Materials[31])
	}
	if user.Materials[32] != 0 || user.Materials[33] != 0 {
		t.Fatal("candidate claim granted a reward for a non-claimable mission")
	}
	if user.Missions[200001].MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatal("candidate claim changed the in-progress mission state")
	}
}

func TestMissionServiceRegistersReceiveMissionRewardsByCategory(t *testing.T) {
	for _, method := range pb.MissionService_ServiceDesc.Methods {
		if method.MethodName == "ReceiveMissionRewardsByCategory" {
			return
		}
	}
	t.Fatal("ReceiveMissionRewardsByCategory RPC is not registered")
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
