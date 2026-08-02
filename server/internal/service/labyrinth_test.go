package service

import (
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestLabyrinthStageClearUsesStoredQuestState(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[100] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
	if labyrinthStageCleared(user, []int32{100, 200}) {
		t.Fatal("partially cleared stage was accepted")
	}
	user.Quests[200] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
	if !labyrinthStageCleared(user, []int32{100, 200}) {
		t.Fatal("fully cleared stage was rejected")
	}
}
