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

func TestQuestSceneChoiceProjectionUsesEffectKeys(t *testing.T) {
	user := store.UserState{
		UserId: 2,
		QuestSceneChoices: map[int32]store.QuestSceneChoiceState{
			1: {QuestSceneChoiceGroupingId: 1, QuestSceneChoiceEffectId: 3, LatestVersion: 456},
		},
		QuestSceneChoiceHistory: map[int32]store.QuestSceneChoiceHistoryState{
			3: {QuestSceneChoiceEffectId: 3, ChoiceDatetime: 123, LatestVersion: 456},
		},
	}

	var current []map[string]any
	if err := json.Unmarshal([]byte(projectTable("IUserQuestSceneChoice", user)), &current); err != nil {
		t.Fatal(err)
	}
	if len(current) != 1 || current[0]["questSceneChoiceGroupingId"] != float64(1) || current[0]["questSceneChoiceEffectId"] != float64(3) {
		t.Fatalf("current choice projection = %v", current)
	}
	for _, unsupported := range []string{"questSceneId", "questFlowType", "choiceNumber", "choiceDatetime"} {
		if _, exists := current[0][unsupported]; exists {
			t.Fatalf("current choice projection used unsupported field %q: %v", unsupported, current[0])
		}
	}

	var history []map[string]any
	if err := json.Unmarshal([]byte(projectTable("IUserQuestSceneChoiceHistory", user)), &history); err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0]["questSceneChoiceEffectId"] != float64(3) || history[0]["choiceDatetime"] != float64(123) {
		t.Fatalf("choice history projection = %v", history)
	}
	for _, unsupported := range []string{"questSceneId", "questFlowType", "choiceNumber", "questSceneChoiceGroupingId"} {
		if _, exists := history[0][unsupported]; exists {
			t.Fatalf("choice history projection used unsupported field %q: %v", unsupported, history[0])
		}
	}
}
