package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestGachaEditorExposesTicketTiersAndActivityKeepsOneFamily(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if err := memorydb.Init(path); err != nil {
		t.Fatal(err)
	}
	entries, _, err := masterdata.LoadGachaCatalog(0)
	if err != nil {
		t.Fatal(err)
	}
	config := gacha.DefaultConfig()
	config.EventBanners[329011] = gacha.EventBoxConfig{Boxes: []gacha.BoxConfig{{}, {}}}
	catalog, err := LoadGachaEditorCatalog(path, &masterdata.GachaCatalog{}, &masterdata.WeaponCatalog{}, &masterdata.CostumeCatalog{}, entries, config, "hash", "master", true)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, banner := range catalog.BoxBanners {
		if banner.EventGachaBaseId != 329001 {
			continue
		}
		seen[banner.EventGachaTicketTier] = true
		if banner.RequiredConsumableItemId == 0 || !strings.Contains(strings.ToLower(banner.TicketNames["en"]), banner.EventGachaTicketTier) || !strings.Contains(strings.ToLower(banner.Titles["en"]), banner.EventGachaTicketTier) {
			t.Fatalf("tier is missing ticket/title metadata: %+v", banner)
		}
		want := int32(0)
		if banner.EventGachaTicketTier == "silver" {
			want = 2
		}
		if banner.ConfiguredBoxCount != want {
			t.Fatalf("configured count: %+v", banner)
		}
	}
	if !seen["bronze"] || !seen["silver"] || !seen["gold"] {
		t.Fatalf("editor tiers = %v", seen)
	}
	groups, err := LoadActivityGroups(path, nil, config, entries)
	if err != nil {
		t.Fatal(err)
	}
	baseFound := false
	for _, option := range groups.Options {
		if option.Kind != "event" {
			continue
		}
		if option.ID == 329001 {
			baseFound = true
		}
		if option.ID == 329011 || option.ID == 329021 {
			t.Fatalf("activity duplicated a ticket family: %+v", option)
		}
	}
	if !baseFound {
		t.Fatal("activity lost the base event")
	}
}

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
