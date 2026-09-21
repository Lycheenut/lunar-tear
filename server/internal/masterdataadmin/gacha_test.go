package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestCostumeIconPathUsesCostumeAssetNaming(t *testing.T) {
	costume := masterdata.EntityMCostume{ActorSkeletonId: 8, AssetVariationId: 13}
	if got, want := costumeIconPath(costume), "costume/ch008013/ch008013_standard.png"; got != want {
		t.Fatalf("costumeIconPath() = %q, want %q", got, want)
	}
}

func TestLoadGachaMedalReferencesIncludesLocalizedConsumableNames(t *testing.T) {
	masterDataPath := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(masterDataPath); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	file, err := memorydb.OpenFile(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}
	resolver := newTitleResolver(file, loadLocalizationIndex(masterDataPath))
	medals := loadGachaMedalReferences(file, resolver)
	if len(medals) == 0 {
		t.Fatal("Gacha medal reference list is empty")
	}
	for _, medal := range medals {
		if medal.GachaMedalId != 8151 {
			continue
		}
		if medal.Names["en"] != "Shard (Resolute Dress)" {
			t.Fatalf("Gacha medal 8151 English name = %q", medal.Names["en"])
		}
		return
	}
	t.Fatal("Gacha medal 8151 is missing")
}
