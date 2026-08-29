package userdata

import (
	"encoding/json"
	"testing"

	"lunar-tear/server/internal/store"
)

func TestQuestReplayFlowRewardProjectionUsesClientSchema(t *testing.T) {
	user := store.UserState{
		UserId: 2,
		QuestReplayFlowRewards: map[int32]store.QuestReplayFlowRewardState{
			30330: {
				QuestReplayFlowRewardGroupId: 30330,
				RewardReceiveDatetime:        123,
				LatestVersion:                456,
			},
		},
	}

	var records []map[string]any
	if err := json.Unmarshal([]byte(projectTable("IUserQuestReplayFlowRewardGroup", user)), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0]["receiveDatetime"] != float64(123) {
		t.Fatalf("replay reward projection = %v", records)
	}
	if _, exists := records[0]["rewardReceiveDatetime"]; exists {
		t.Fatalf("replay reward projection used unsupported field: %v", records[0])
	}
}
