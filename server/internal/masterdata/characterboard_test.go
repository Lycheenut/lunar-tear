package masterdata

import (
	"testing"

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
