package memorydb

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestInspectMissionTableNames(t *testing.T) {
	if err := Init(filepath.Join("..", "..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	var names []string
	for name := range tables {
		if strings.Contains(name, "mission") || strings.Contains(name, "quest_type") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		t.Log(name)
	}
}
