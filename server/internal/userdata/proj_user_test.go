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

func TestCampaignAndLoginBonusProjectionUsesClientTables(t *testing.T) {
	user := store.UserState{
		UserId: 7,
		BeginnerCampaign: store.UserBeginnerCampaignState{
			BeginnerCampaignId:       1,
			CampaignRegisterDatetime: 100,
			LatestVersion:            101,
		},
		ComebackCampaign: store.UserComebackCampaignState{
			ComebackCampaignId: 2,
			ComebackDatetime:   200,
			LatestVersion:      201,
		},
		LoginBonuses: map[int32]store.UserLoginBonusState{
			91: {LoginBonusId: 91, CurrentPageNumber: 1, CurrentStampNumber: 1},
			1:  {LoginBonusId: 1, CurrentPageNumber: 2, CurrentStampNumber: 3},
		},
	}

	var beginner []map[string]any
	if err := json.Unmarshal([]byte(projectTable("IUserBeginnerCampaign", user)), &beginner); err != nil {
		t.Fatal(err)
	}
	var comeback []map[string]any
	if err := json.Unmarshal([]byte(projectTable("IUserComebackCampaign", user)), &comeback); err != nil {
		t.Fatal(err)
	}
	var bonuses []struct {
		LoginBonusId int32 `json:"loginBonusId"`
	}
	if err := json.Unmarshal([]byte(projectTable("IUserLoginBonus", user)), &bonuses); err != nil {
		t.Fatal(err)
	}
	if len(beginner) != 1 || len(comeback) != 1 || len(bonuses) != 2 || bonuses[0].LoginBonusId != 1 || bonuses[1].LoginBonusId != 91 {
		t.Fatalf("campaign projections: beginner=%v comeback=%v bonuses=%v", beginner, comeback, bonuses)
	}

	before := &store.UserState{LoginBonuses: map[int32]store.UserLoginBonusState{}}
	after := store.CloneUserState(*before)
	after.BeginnerCampaign = user.BeginnerCampaign
	after.ComebackCampaign = user.ComebackCampaign
	after.LoginBonuses[1] = user.LoginBonuses[1]
	changed := ChangedTables(before, &after)
	for _, table := range []string{"IUserBeginnerCampaign", "IUserComebackCampaign", "IUserLoginBonus"} {
		if !slices.Contains(changed, table) {
			t.Fatalf("changed campaign tables = %v, missing %s", changed, table)
		}
	}
	if keys := keyFieldsForTable("IUserLoginBonus"); !slices.Equal(keys, []string{"userId", "loginBonusId"}) {
		t.Fatalf("login bonus keys = %v", keys)
	}
}

func TestEmptyCampaignProjectionsUseJSONArrays(t *testing.T) {
	user := store.UserState{}
	for _, table := range []string{"IUserBeginnerCampaign", "IUserComebackCampaign"} {
		if got := projectTable(table, user); got != "[]" {
			t.Errorf("%s projection = %s, want []", table, got)
		}
	}
}
