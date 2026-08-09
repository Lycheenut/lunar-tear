package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLocalizedText(t *testing.T) {
	entries := parseLocalizedText("# comment\r\nexample.title:Title: With Detail\r\nignored\r\n")
	if got, want := entries["example.title"], "Title: With Detail"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
}

func TestReadInstalledEventQuestTextAsset(t *testing.T) {
	bundleRoot := filepath.Join("..", "..", "assets", "revisions", "0", "assetbundle")
	path := filepath.Join(bundleRoot, "text", "en", "quest", "event_quest.assetbundle")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository text assets are not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	entries, err := readTextAssetBundle(path, bundleRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := entries["quest.event.chapter_title.500"], "Record: The Cage of Reincarnation"; got != want {
		t.Fatalf("event title = %q, want %q", got, want)
	}
}

func TestLoadAddsInstalledLocalizedTitles(t *testing.T) {
	masterDataPath := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(masterDataPath); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master data is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	index := loadLocalizationIndex(masterDataPath)
	for _, key := range []string{
		"quest.event.chapter_title.500",
		"gacha.title.limited_45",
		"consumable_item.name.110004",
		"important_item.name.100001",
	} {
		if index["en"][key] == "" {
			t.Fatalf("English localization index is missing %s", key)
		}
	}
	catalog, err := Load(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.DefaultLanguage != "en" {
		t.Fatalf("default language = %q, want en", catalog.DefaultLanguage)
	}
	for _, table := range catalog.Tables {
		if table.Name != "m_event_quest_chapter" {
			continue
		}
		for _, row := range table.Rows {
			if row.Titles["en"] != "" {
				return
			}
		}
	}
	t.Fatal("event quest catalog has no English title")
}

func TestLoadAddsContentFootnotes(t *testing.T) {
	masterDataPath := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(masterDataPath); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master data is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}
	enhanceTargetGroups := map[string]bool{}
	enhanceTargetRows := map[string][]map[string]string{}
	questTargetGroups := map[string]bool{}
	questTargetRows := map[string][]map[string]string{}
	questMagnitudeGroups := map[string]bool{}
	maintenanceGroups := map[string]bool{}
	for _, table := range catalog.Tables {
		for _, row := range table.Rows {
			switch table.Name {
			case "m_enhance_campaign_target_group":
				groupID := row.Values["EnhanceCampaignTargetGroupId"]
				enhanceTargetGroups[groupID] = true
				enhanceTargetRows[groupID] = append(enhanceTargetRows[groupID], row.Values)
			case "m_quest_campaign_target_group":
				groupID := row.Values["QuestCampaignTargetGroupId"]
				questTargetGroups[groupID] = true
				questTargetRows[groupID] = append(questTargetRows[groupID], row.Values)
			case "m_quest_campaign_effect_group":
				if row.Values["QuestCampaignEffectType"] != "5" {
					questMagnitudeGroups[row.Values["QuestCampaignEffectGroupId"]] = true
				}
			case "m_maintenance_group":
				maintenanceGroups[row.Values["MaintenanceGroupId"]] = true
			}
		}
	}
	checked := map[string]int{}
	for _, table := range catalog.Tables {
		for _, row := range table.Rows {
			switch table.Name {
			case "m_enhance_campaign", "m_quest_campaign":
				if row.Titles["en"] == "" {
					continue
				}
				for language, title := range row.Titles {
					if strings.Contains(title, "{0}") {
						t.Fatalf("%s row %d has unresolved %s title parameter: %q", table.Name, row.Index, language, title)
					}
				}
				if len(row.ContentFootnotes) == 0 {
					t.Fatalf("%s row %d has no content footnote: %v", table.Name, row.Index, row.Values)
				}
				if table.Name == "m_enhance_campaign" && enhanceTargetGroups[row.Values["EnhanceCampaignTargetGroupId"]] && len(row.ContentFootnotes) < 2 {
					t.Fatalf("%s row %d has no target-name footnote: %v; targets: %v", table.Name, row.Index, row.Values, enhanceTargetRows[row.Values["EnhanceCampaignTargetGroupId"]])
				}
				if table.Name == "m_quest_campaign" && questTargetGroups[row.Values["QuestCampaignTargetGroupId"]] && questMagnitudeGroups[row.Values["QuestCampaignEffectGroupId"]] && len(row.ContentFootnotes) < 2 {
					t.Fatalf("%s row %d has no dungeon-name footnote: %v; targets: %v", table.Name, row.Index, row.Values, questTargetRows[row.Values["QuestCampaignTargetGroupId"]])
				}
				checked[table.Name]++
			case "m_maintenance":
				if maintenanceGroups[row.Values["MaintenanceGroupId"]] && row.Titles["en"] == "" {
					t.Fatalf("%s row %d has no affected-API title", table.Name, row.Index)
				}
				if len(row.ContentFootnotes) != 0 {
					t.Fatalf("%s row %d still has content footnotes", table.Name, row.Index)
				}
				checked[table.Name]++
			case "m_shop_item_cell_term":
				if len(row.ShopRelations) != 0 && len(row.ContentFootnotes) == 0 {
					t.Fatalf("%s row %d has shops but no content footnote", table.Name, row.Index)
				}
				checked[table.Name]++
			}
		}
	}
	for _, table := range []string{"m_enhance_campaign", "m_quest_campaign", "m_maintenance", "m_shop_item_cell_term"} {
		if checked[table] == 0 {
			t.Fatalf("did not check any rows from %s", table)
		}
	}
}
