package userdata

import (
	"encoding/json"
	"slices"
	"testing"

	"lunar-tear/server/internal/store"
)

func TestMissionCompletionProgressProjectionUsesClientSchema(t *testing.T) {
	user := store.UserState{UserId: 7, Missions: map[int32]store.UserMissionState{
		4206: {MissionId: 4206, ProgressValue: 1, LatestVersion: 99},
	}}

	var records []struct {
		UserId        int64 `json:"userId"`
		MissionId     int32 `json:"missionId"`
		ProgressValue int64 `json:"progressValue"`
		LatestVersion int64 `json:"latestVersion"`
	}
	if err := json.Unmarshal([]byte(projectTable("IUserMissionCompletionProgress", user)), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].UserId != 7 || records[0].MissionId != 4206 || records[0].ProgressValue != 1 || records[0].LatestVersion != 99 {
		t.Fatalf("mission completion progress projection = %+v", records)
	}
}

func TestMissionChangesRefreshBothClientMissionTables(t *testing.T) {
	before := &store.UserState{Missions: map[int32]store.UserMissionState{}}
	after := &store.UserState{Missions: map[int32]store.UserMissionState{
		4206: {MissionId: 4206, ProgressValue: 1, LatestVersion: 99},
	}}

	changed := ChangedTables(before, after)
	if !slices.Contains(changed, "IUserMission") || !slices.Contains(changed, "IUserMissionCompletionProgress") {
		t.Fatalf("changed mission tables = %v", changed)
	}
	if keys := keyFieldsForTable("IUserMissionCompletionProgress"); !slices.Equal(keys, []string{"userId", "missionId"}) {
		t.Fatalf("mission completion progress keys = %v", keys)
	}
}
