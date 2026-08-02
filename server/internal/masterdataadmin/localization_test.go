package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
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
	if index["en"]["quest.event.chapter_title.500"] == "" {
		t.Fatal("English localization index is missing event chapter title 500")
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
