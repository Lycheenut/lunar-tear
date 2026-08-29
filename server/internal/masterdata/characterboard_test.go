package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/store"
)

func TestCharacterBoardReleaseBitPreventsRepeat(t *testing.T) {
	board := store.CharacterBoardState{PanelReleaseBit2: 1 << 2}
	if !IsCharacterBoardPanelReleased(board, 35) {
		t.Fatal("released panel not detected")
	}
	if IsCharacterBoardPanelReleased(board, 34) {
		t.Fatal("unreleased panel detected")
	}
}

func TestCharacterBoardMissionOptionsUseMissionOrder(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadCharacterBoardCatalog()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		boardId int32
		want    int32
	}{
		{name: "Levania stone tower", boardId: 100101, want: 310021},
		{name: "Levania cursed god", boardId: 100104, want: 310022},
		{name: "063y stone tower", boardId: 100701, want: 310011},
		{name: "Noelle cursed god", boardId: 100604, want: 310020},
		{name: "Fio stone tower", boardId: 101201, want: 310023},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := catalog.MissionOptionByBoardId[test.boardId]; got != test.want {
				t.Fatalf("board %d maps to mission option %d, want %d", test.boardId, got, test.want)
			}
		})
	}
}
