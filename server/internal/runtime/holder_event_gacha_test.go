package runtime_test

import (
	"os"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/runtime"
)

func TestHolderLoadsAndReloadsEventTicketTiers(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	masterPath, configPath := filepath.Join(dir, "master.bin.e"), filepath.Join(dir, "gacha.json")
	if err := os.WriteFile(masterPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	config := gacha.DefaultConfig()
	for _, id := range []int32{329001, 329011, 329021} {
		config.EventBanners[id] = gacha.EventBoxConfig{Boxes: []gacha.BoxConfig{{GroupWeights: gacha.BoxGroupWeights{Limited: 10000}, LimitedRewards: []gacha.BoxRewardConfig{{PossessionType: 6, PossessionId: 1, Count: 1, MaxCount: 1, Jackpot: true}}}}}
	}
	config.EventSchedules = map[int32]gacha.EventSchedule{329001: {StartDatetime: 1000, EndDatetime: 2000}}
	writeConfig := func() {
		t.Helper()
		raw, _, err := gacha.EncodeConfig(config)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(configPath, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeConfig()
	holder, err := runtime.NewHolderWithGachaConfig(masterPath, configPath)
	if err != nil {
		t.Fatal(err)
	}
	check := func(silverBoxes int32) {
		t.Helper()
		found := 0
		for _, entry := range holder.Get().GachaEntries {
			if entry.EventGachaBaseId != 329001 {
				continue
			}
			found++
			want := int32(1)
			if entry.EventGachaTicketTier == "silver" {
				want = silverBoxes
			}
			if entry.BoxCount != want || entry.StartDatetime != 1000 || entry.EndDatetime != 2000 {
				t.Fatalf("loaded tier: %+v", entry)
			}
		}
		if found != 3 {
			t.Fatalf("loaded %d tiers", found)
		}
	}
	check(1)
	delete(config.EventBanners, 329011)
	writeConfig()
	if err := holder.Reload(); err != nil {
		t.Fatal(err)
	}
	check(0)
	previous := holder.Get()
	config.EventBanners[999999] = config.EventBanners[329001]
	writeConfig()
	if err := holder.Reload(); err == nil {
		t.Fatal("invalid tier accepted on reload")
	}
	if holder.Get() != previous {
		t.Fatal("failed reload replaced live catalogs")
	}
}
