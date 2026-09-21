package activitygroup

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"lunar-tear/server/internal/gacha"
)

func TestReadConfigDistinguishesMissingAndSavedEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activity-groups.json")
	config, hash, err := ReadConfig(path)
	if err != nil || config != nil || hash != gacha.ContentHash(nil) {
		t.Fatalf("missing config = %v, %q, %v", config, hash, err)
	}
	empty := &Config{Version: 1, Units: []ActivityUnit{}, Groups: []ActivityGroup{}}
	raw, err := EncodeConfig(empty)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	config, hash, err = ReadConfig(path)
	if err != nil || !reflect.DeepEqual(config, empty) || hash != gacha.ContentHash(raw) {
		t.Fatalf("saved empty config = %v, %q, %v", config, hash, err)
	}
	for _, invalid := range []string{`null`, `{}`, `{"version":2}`, `{"version":1,"banners":{}}`, `{"version":1} {}`, `{"version":1,"eventSchedules":{}}`} {
		if err := os.WriteFile(path, []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := ReadConfig(path); err == nil {
			t.Fatalf("invalid config accepted: %s", invalid)
		}
	}
}
