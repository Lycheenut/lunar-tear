package runtime_test

import (
	"errors"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"testing"

	"lunar-tear/server/internal/activitygroup"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdataadmin"
	"lunar-tear/server/internal/questdrop"
	"lunar-tear/server/internal/runtime"
)

func TestInstallAndReloadCurrentMasterDataCandidate(t *testing.T) {
	source := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	original, err := os.ReadFile(source)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	target := filepath.Join(directory, "current.bin.e")
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatal(err)
	}

	catalog, err := masterdataadmin.Load(target)
	if err != nil {
		t.Fatal(err)
	}
	var request masterdataadmin.UpdateRequest
	request.ExpectedVersion = catalog.Version
	for _, table := range catalog.Tables {
		if len(table.Rows) == 0 || len(table.Pairs) == 0 {
			continue
		}
		endField := table.Pairs[0].End
		end := table.Rows[0].Times[endField]
		if end > 0 {
			request.Changes = []masterdataadmin.Change{{
				Table: table.Name,
				Row:   table.Rows[0].Index,
				Field: endField,
				Value: end + 1000,
			}}
			break
		}
	}
	if len(request.Changes) == 0 {
		t.Fatal("no suitable schedule row found")
	}
	candidate, result, err := masterdataadmin.BuildUpdate(target, request)
	if err != nil {
		t.Fatal(err)
	}
	candidatePath := filepath.Join(directory, "candidate.bin.e")
	if err := os.WriteFile(candidatePath, candidate, 0o600); err != nil {
		t.Fatal(err)
	}

	holder, err := runtime.NewHolder(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := holder.InstallAndReload(candidatePath); err != nil {
		t.Fatal(err)
	}
	if holder.Get() == nil {
		t.Fatal("holder did not publish candidate catalogs")
	}
	if _, err := os.Stat(candidatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("candidate was not atomically moved: %v", err)
	}
	installed, err := masterdataadmin.Load(target)
	if err != nil {
		t.Fatal(err)
	}
	if installed.Version != result.Version {
		t.Fatalf("installed version = %s, want %s", installed.Version, result.Version)
	}
}

func TestInstallGachaConfigPublishesValidatedSnapshot(t *testing.T) {
	source := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	original, err := os.ReadFile(source)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	masterDataPath := filepath.Join(directory, "current.bin.e")
	configPath := filepath.Join(directory, "gacha.json")
	if err := os.WriteFile(masterDataPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	holder, err := runtime.NewHolderWithGachaConfig(masterDataPath, configPath)
	if err != nil {
		t.Fatal(err)
	}
	before := holder.Get()
	if len(before.PremiumGacha.Banners) == 0 {
		t.Fatal("missing Gacha config did not build default standard banner pools")
	}
	editor, err := masterdataadmin.LoadGachaEditorCatalog(
		masterDataPath,
		before.GachaPool,
		before.Weapon,
		before.Costume,
		before.GachaEntries,
		before.GachaConfig,
		before.GachaConfigHash,
		before.MasterDataHash,
		before.GachaConfigExists,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(editor.Weapons), len(before.GachaPool.EligibleWeaponById); got != want {
		t.Fatalf("editor weapon count = %d, want %d eligible weapons", got, want)
	}
	if len(editor.BoxBanners) == 0 {
		t.Fatal("Gacha editor did not expose Chapter/Event box configuration")
	}
	if len(editor.Config.ChapterBanners) != 0 {
		t.Fatal("Gacha editor synthesized Chapter reward configuration")
	}
	config := editor.Config
	config.SourceMasterDataHash = before.MasterDataHash
	for weaponID := range before.GachaPool.ConfigurableWeaponById {
		availability := gacha.AvailabilityEvent
		if _, eligible := before.GachaPool.EligibleWeaponById[weaponID]; eligible {
			availability = gacha.AvailabilityStandard
		}
		config.Weapons[weaponID] = gacha.WeaponConfig{Availability: availability}
	}
	config = gacha.ConfigWithoutAutomaticEventWeapons(config, before.GachaPool)
	for _, id := range []int32{329001, 329011, 329021} {
		config.EventBanners[id] = gacha.EventBoxConfig{Boxes: []gacha.BoxConfig{{
			GroupWeights:   gacha.BoxGroupWeights{Limited: 10000},
			LimitedRewards: []gacha.BoxRewardConfig{{PossessionType: 6, PossessionId: 1, Count: 1, MaxCount: 1, Jackpot: true}},
		}}}
	}
	encoded, expectedInstalledHash, err := gacha.EncodeConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	candidatePath := filepath.Join(directory, "candidate.json")
	if err := os.WriteFile(candidatePath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := holder.InstallGachaConfig(candidatePath, before.GachaConfigHash); err != nil {
		t.Fatal(err)
	}

	after := holder.Get()
	for _, id := range []int32{329001, 329011, 329021} {
		if event := after.GachaConfig.EventBanners[id]; len(event.Boxes) != 1 {
			t.Fatalf("published event tier %d = %+v", id, event)
		}
		found := false
		for _, entry := range after.GachaEntries {
			if entry.GachaId == id {
				found = entry.BoxCount == 1
			}
		}
		if !found {
			t.Fatalf("published tier %d is not available in the runtime catalog", id)
		}
	}
	if !after.GachaConfigExists {
		t.Fatal("published snapshot does not report an installed Gacha config")
	}
	if after.GachaConfigHash != expectedInstalledHash {
		t.Fatalf("Gacha config hash = %q, want %q", after.GachaConfigHash, expectedInstalledHash)
	}
	if len(after.PremiumGacha.Banners) == 0 {
		t.Fatal("published snapshot has no premium banner pools")
	}
	if _, err := os.Stat(candidatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("candidate was not atomically moved: %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("installed Gacha config is missing: %v", err)
	}

	conflictingCandidatePath := filepath.Join(directory, "conflicting.json")
	if err := os.WriteFile(conflictingCandidatePath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := holder.InstallGachaConfig(conflictingCandidatePath, before.GachaConfigHash); !errors.Is(err, runtime.ErrGachaConfigConflict) {
		t.Fatalf("stale publish error = %v, want conflict", err)
	}
}

func TestInstallActivityConfigPublishesIndependentConfigAndSchedule(t *testing.T) {
	source := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	original, err := os.ReadFile(source)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	masterDataPath := filepath.Join(directory, "current.bin.e")
	configPath := filepath.Join(directory, "gacha.json")
	activityPath := filepath.Join(directory, "activity-groups.json")
	originalActivity := []byte(`{"version":1,"units":[],"groups":[]}`)
	if err := os.WriteFile(activityPath, originalActivity, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(masterDataPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := runtime.NewHolderWithConfigs(masterDataPath, configPath, "", activityPath)
	if err != nil {
		t.Fatal(err)
	}
	before := holder.Get()
	config := gacha.DefaultConfig()
	config.Banners[588] = gacha.BannerConfig{
		BannerAssetName: "limited_588",
		StartDatetime:   gacha.DefaultBannerStartDatetime,
		EndDatetime:     gacha.DefaultBannerEndDatetime,
	}
	groups, err := masterdataadmin.GenerateActivityGroups(masterDataPath, nil, config, before.GachaEntries)
	if err != nil {
		t.Fatal(err)
	}
	groupCatalog, err := masterdataadmin.LoadActivityGroups(masterDataPath, groups, config, before.GachaEntries)
	if err != nil {
		t.Fatal(err)
	}
	masterCandidateRaw, config, _, err := masterdataadmin.BuildActivitySchedule(masterDataPath, groups, config, groupCatalog, "premium:588", gacha.DefaultBannerStartDatetime, gacha.DefaultBannerEndDatetime)
	if err != nil {
		t.Fatal(err)
	}
	config.SourceMasterDataHash = gacha.ContentHash(masterCandidateRaw)
	configCandidateRaw, configHash, err := gacha.EncodeConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	masterCandidatePath := filepath.Join(directory, "master-candidate.bin.e")
	configCandidatePath := filepath.Join(directory, "gacha-candidate.json")
	if err := os.WriteFile(masterCandidatePath, masterCandidateRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configCandidatePath, configCandidateRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	activityRaw, err := activitygroup.EncodeConfig(groups)
	if err != nil {
		t.Fatal(err)
	}
	activityCandidatePath := filepath.Join(directory, "activity-candidate.json")
	if err := os.WriteFile(activityCandidatePath, []byte(`{"version":99}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := holder.InstallActivityConfig(activityCandidatePath, configCandidatePath, masterCandidatePath, before.ActivityConfigHash, before.GachaConfigHash, before.MasterDataHash); err == nil {
		t.Fatal("invalid activity file accepted")
	}
	if holder.Get() != before {
		t.Fatal("failed validation published a snapshot")
	}
	if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed validation installed Gacha config")
	}
	if err := os.WriteFile(activityCandidatePath, activityRaw, 0600); err != nil {
		t.Fatal(err)
	}
	if stdruntime.GOOS == "windows" {
		// Holding the activity file open rejects its replacement after the
		// master-data and Gacha candidates have already been installed.
		locked, err := os.Open(activityPath)
		if err != nil {
			t.Fatal(err)
		}
		installErr := holder.InstallActivityConfig(activityCandidatePath, configCandidatePath, masterCandidatePath, before.ActivityConfigHash, before.GachaConfigHash, before.MasterDataHash)
		_ = locked.Close()
		if installErr == nil {
			t.Fatal("replaced locked activity file")
		}
		if holder.Get() != before {
			t.Fatal("failed installation published a snapshot")
		}
		if hash, err := gacha.FileHash(masterDataPath); err != nil || hash != before.MasterDataHash {
			t.Fatalf("master data was not rolled back: %v", err)
		}
		if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("new Gacha config was not removed on rollback")
		}
		if hash, err := gacha.FileHash(activityPath); err != nil || hash != before.ActivityConfigHash {
			t.Fatalf("failed installation changed activity config: %v", err)
		}
		if err := os.WriteFile(masterCandidatePath, masterCandidateRaw, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(configCandidatePath, configCandidateRaw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := holder.InstallActivityConfig(activityCandidatePath, configCandidatePath, masterCandidatePath, before.ActivityConfigHash, before.GachaConfigHash, before.MasterDataHash); err != nil {
		t.Fatal(err)
	}
	if err := holder.Reload(); err != nil {
		t.Fatal(err)
	}
	after := holder.Get()
	if after.ActivityConfigHash != gacha.ContentHash(activityRaw) || len(after.ActivityConfig.Groups) != len(groups.Groups) {
		t.Fatal("independent activity config missing after reload")
	}
	if after.GachaConfigHash != configHash || after.MasterDataHash != config.SourceMasterDataHash {
		t.Fatalf("published hashes = Gacha %q, master %q", after.GachaConfigHash, after.MasterDataHash)
	}
	if got := after.GachaConfig.Banners[588]; got.StartDatetime != gacha.DefaultBannerStartDatetime || got.EndDatetime != gacha.DefaultBannerEndDatetime {
		t.Fatalf("published Gacha schedule = %+v", got)
	}
	catalog, err := masterdataadmin.LoadTable(masterDataPath, "m_mom_banner")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range catalog.Tables {
		if table.Name != "m_mom_banner" {
			continue
		}
		for _, row := range table.Rows {
			if row.Values["BannerAssetName"] == "limited_588" {
				if row.Times["StartDatetime"] != gacha.DefaultBannerStartDatetime || row.Times["EndDatetime"] != gacha.DefaultBannerEndDatetime {
					t.Fatalf("installed MomBanner schedule = %d..%d", row.Times["StartDatetime"], row.Times["EndDatetime"])
				}
				return
			}
		}
	}
	t.Fatal("installed limited_588 MomBanner row was not found")
}

func TestInstallQuestDropConfigPublishesWeightedPools(t *testing.T) {
	source := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	original, err := os.ReadFile(source)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	masterDataPath := filepath.Join(directory, "current.bin.e")
	configPath := filepath.Join(directory, "quest_drops.json")
	if err := os.WriteFile(masterDataPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	holder, err := runtime.NewHolderWithConfigs(masterDataPath, "", configPath, "")
	if err != nil {
		t.Fatal(err)
	}
	before := holder.Get()
	var questID, rewardID int32
	for candidateQuestID, quest := range before.Quest.QuestById {
		pool := before.Quest.PickupRewardIdsByGroupId[quest.QuestPickupRewardGroupId]
		if len(pool) > 0 {
			questID, rewardID = candidateQuestID, pool[0]
			break
		}
	}
	if questID == 0 || rewardID == 0 {
		t.Fatal("master data has no quest pickup reward")
	}
	config := questdrop.DefaultConfig()
	config.SourceMasterDataHash = before.MasterDataHash
	config.Quests[questID] = questdrop.QuestConfig{Rewards: []questdrop.Reward{{
		BattleDropRewardID: rewardID,
		Weight:             7,
	}}}
	encoded, expectedInstalledHash, err := questdrop.EncodeConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	candidatePath := filepath.Join(directory, "candidate.json")
	if err := os.WriteFile(candidatePath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := holder.InstallQuestDropConfig(candidatePath, before.QuestDropConfigHash); err != nil {
		t.Fatal(err)
	}

	after := holder.Get()
	if !after.QuestDropConfigExists || after.QuestDropConfigHash != expectedInstalledHash {
		t.Fatalf("installed quest drop config state = exists %v hash %q", after.QuestDropConfigExists, after.QuestDropConfigHash)
	}
	pool := after.QuestHandler.DropRewardsByQuestID[questID]
	if len(pool) != 1 || pool[0].BattleDropRewardID != rewardID || pool[0].Weight != 7 {
		t.Fatalf("published quest drop pool = %+v", pool)
	}
	if _, err := os.Stat(candidatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("candidate was not atomically moved: %v", err)
	}

	conflictingCandidatePath := filepath.Join(directory, "conflicting.json")
	if err := os.WriteFile(conflictingCandidatePath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := holder.InstallQuestDropConfig(conflictingCandidatePath, before.QuestDropConfigHash); !errors.Is(err, runtime.ErrQuestDropConfigConflict) {
		t.Fatalf("stale publish error = %v, want conflict", err)
	}
}
