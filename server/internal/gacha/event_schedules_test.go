package gacha

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEventSchedulesPersistInGachaConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gacha.json")
	for _, schedules := range []map[int32]EventSchedule{
		nil, {}, {329001: {StartDatetime: 1000, EndDatetime: 2000}},
	} {
		config := DefaultConfig()
		config.EventSchedules = schedules
		raw, hash, err := EncodeConfig(config)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		loaded, loadedHash, exists, err := ReadConfig(path)
		if err != nil || !exists || loadedHash != hash || !reflect.DeepEqual(loaded.EventSchedules, schedules) {
			t.Fatalf("Event schedules did not round trip: %v", err)
		}
	}
	for _, raw := range []string{
		`{"eventSchedules":{"0":{"startDatetime":1000,"endDatetime":2000}}}`,
		`{"eventSchedules":{"329001":{"startDatetime":-1,"endDatetime":2000}}}`,
		`{"eventSchedules":{"329001":{"startDatetime":2000,"endDatetime":1000}}}`,
	} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := ReadConfig(path); err == nil {
			t.Fatalf("invalid Event schedule accepted: %s", raw)
		}
	}
}
