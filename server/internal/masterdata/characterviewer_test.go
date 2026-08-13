package masterdata

import (
	"slices"
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestCharacterViewerNewlyReleasedFieldIdsExcludePersistedFields(t *testing.T) {
	catalog := &CharacterViewerCatalog{fields: []characterViewerFieldEntry{
		{FieldId: 1},
		{FieldId: 2, RequiredQuestId: 100},
		{FieldId: 3, RequiredQuestId: 200},
	}}
	user := store.SeedUserState(1, "character-viewer", 1, model.ClientPlatform{})
	user.CharacterViewerFields[1] = store.CharacterViewerFieldState{CharacterViewerFieldId: 1}
	user.Quests[100] = store.UserQuestState{QuestId: 100, QuestStateType: model.UserQuestStateTypeCleared}

	if got := catalog.NewlyReleasedFieldIds(*user); !slices.Equal(got, []int32{2}) {
		t.Fatalf("newly released fields = %v, want [2]", got)
	}
}
