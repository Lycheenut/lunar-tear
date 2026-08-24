package masterdataadmin

import "testing"

func TestQuestDropCharacterQuestStage(t *testing.T) {
	tests := []struct {
		nameTextID int32
		category   int32
		level      int32
	}{
		{110001, 1, 1}, {110003, 1, 2}, {110005, 1, 3},
		{110002, 2, 1}, {110004, 2, 2}, {110006, 2, 3},
		{110008, 3, 1}, {110009, 3, 2}, {110010, 3, 3},
		{110011, 0, 0}, {110012, 0, 0}, {110013, 0, 0},
		{0, 0, 0},
	}
	for _, test := range tests {
		category, level := questDropCharacterQuestStage(test.nameTextID)
		if category != test.category || level != test.level {
			t.Fatalf("NameQuestTextId %d = category %d level %d, want category %d level %d",
				test.nameTextID, category, level, test.category, test.level)
		}
	}
}
