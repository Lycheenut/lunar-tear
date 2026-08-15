package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestStarterWeaponsCannotBeDiscarded(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	weapons, err := memorydb.ReadTable[EntityMWeapon]("m_weapon")
	if err != nil {
		t.Fatal(err)
	}

	starterWeaponIds := map[int32]bool{100001: false, 100011: false, 100021: false}
	for _, weapon := range weapons {
		if _, ok := starterWeaponIds[weapon.WeaponId]; ok {
			starterWeaponIds[weapon.WeaponId] = weapon.IsRestrictDiscard
		}
	}
	for weaponId, restricted := range starterWeaponIds {
		if !restricted {
			t.Errorf("starter weapon %d is missing or can be discarded", weaponId)
		}
	}
}
