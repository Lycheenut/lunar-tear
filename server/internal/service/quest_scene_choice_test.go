package service

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestApplyQuestSceneChoiceReplacesGroupingAndKeepsFirstHistory(t *testing.T) {
	user := store.SeedUserState(2, "ending-player", 2, model.ClientPlatform{})
	effect1 := masterdata.EntityMQuestSceneChoiceEffect{QuestSceneChoiceEffectId: 1, QuestSceneChoiceGroupingId: 1}
	effect2 := masterdata.EntityMQuestSceneChoiceEffect{QuestSceneChoiceEffectId: 2, QuestSceneChoiceGroupingId: 1}

	applyQuestSceneChoice(user, effect1, 100)
	applyQuestSceneChoice(user, effect2, 200)
	applyQuestSceneChoice(user, effect1, 300)

	current := user.QuestSceneChoices[1]
	if len(user.QuestSceneChoices) != 1 || current.QuestSceneChoiceEffectId != 1 || current.LatestVersion != 300 {
		t.Fatalf("current choice = %+v in %d groups, want effect 1 at version 300", current, len(user.QuestSceneChoices))
	}
	if len(user.QuestSceneChoiceHistory) != 2 {
		t.Fatalf("choice history count = %d, want 2", len(user.QuestSceneChoiceHistory))
	}
	first := user.QuestSceneChoiceHistory[1]
	if first.ChoiceDatetime != 100 || first.LatestVersion != 100 {
		t.Fatalf("first choice history = %+v, want original time/version 100", first)
	}
}
