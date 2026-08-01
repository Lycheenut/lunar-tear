package masterdata

import "testing"

func TestLimitContentKeepsAllRestrictionTypesForTarget(t *testing.T) {
	catalog := &LimitContentCatalog{
		ContentsByChapter: map[int32][]EntityMEventQuestLimitContent{1: {{StartDatetime: 1, EndDatetime: 100, EventQuestLimitContentDeckRestrictionId: 1}}},
		RestrictionsById:  map[int32][]EntityMEventQuestLimitContentDeckRestriction{1: {{EventQuestLimitContentDeckRestrictionTargetId: 1, StartDatetime: 1, EndDatetime: 100}}},
		TargetTypesById:   map[int32][]int32{1: {1, 2}},
	}
	types := catalog.ActiveRestrictionTypes(1, 50)
	if len(types) != 2 || types[0] != 1 || types[1] != 2 {
		t.Fatalf("restriction types = %v", types)
	}
}
