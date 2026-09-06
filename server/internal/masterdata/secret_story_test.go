package masterdata

import (
	"path/filepath"
	"slices"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
)

func TestSecretStoryQuestConditionsHaveHiddenMissionMappings(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	parts, err := LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := LoadConditionResolver()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for id, condition := range resolver.conditionsById {
		if id < 500000 || id >= 600000 || condition.EvaluateConditionFunctionType != int32(model.EvaluateConditionFunctionTypeQuestMissionClear) {
			continue
		}
		values := resolver.valuesByGroupId[condition.EvaluateConditionValueGroupId]
		if len(values) != 2 {
			t.Fatalf("condition %d values = %v", id, values)
		}
		questId, missionId := int32(values[0].Value), int32(values[1].Value)
		if !slices.Contains(catalog.InvisibleMissionIdsByQuestId[questId], missionId) {
			t.Errorf("Secret Story condition %d cannot record quest %d mission %d", id, questId, missionId)
		}
		if slices.Contains(catalog.MissionIdsByQuestId[questId], missionId) {
			t.Errorf("hidden mission %d leaked into quest %d stars", missionId, questId)
		}
		if _, ok := catalog.MissionById[missionId]; !ok {
			t.Errorf("missing hidden mission %d", missionId)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no Secret Story quest conditions checked")
	}
	t.Logf("checked %d Secret Story conditions across %d quests with hidden missions", checked, len(catalog.InvisibleMissionIdsByQuestId))
}
