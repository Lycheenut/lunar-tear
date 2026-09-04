package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestCostumeIconPathUsesCostumeAssetNaming(t *testing.T) {
	costume := masterdata.EntityMCostume{ActorSkeletonId: 8, AssetVariationId: 13}
	if got, want := costumeIconPath(costume), "costume/ch008013/ch008013_standard.png"; got != want {
		t.Fatalf("costumeIconPath() = %q, want %q", got, want)
	}
}

func TestBuildGachaMomBannerUpdateSynchronizesMatchingAsset(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	config := gacha.DefaultConfig()
	config.Banners[588] = gacha.BannerConfig{
		BannerAssetName: "limited_588",
		StartDatetime:   gacha.DefaultBannerStartDatetime,
		EndDatetime:     gacha.DefaultBannerEndDatetime,
	}
	candidate, result, err := BuildGachaMomBannerUpdate(path, config)
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedRows < 1 || result.ChangedCells < 2 {
		t.Fatalf("MomBanner update = %d rows/%d cells, want at least 1 row/2 cells", result.ChangedRows, result.ChangedCells)
	}
	file, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows, exists, err := file.TableRows("m_mom_banner")
	if err != nil || !exists {
		t.Fatalf("read candidate MomBanner: exists=%v err=%v", exists, err)
	}
	spec, _ := tableSpecByName("m_mom_banner")
	assetField, _ := findField(spec, "BannerAssetName")
	startField, _ := findField(spec, "StartDatetime")
	endField, _ := findField(spec, "EndDatetime")
	for _, row := range rows {
		if row[assetField.Index] != "limited_588" {
			continue
		}
		if row[startField.Index] != gacha.DefaultBannerStartDatetime || row[endField.Index] != gacha.DefaultBannerEndDatetime {
			t.Fatalf("synchronized MomBanner times = %v..%v", row[startField.Index], row[endField.Index])
		}
		return
	}
	t.Fatal("limited_588 MomBanner row was not found")
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
