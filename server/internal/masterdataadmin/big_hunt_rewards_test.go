package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestCurrentBigHuntRewardConfigIsValid(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	file, err := memorydb.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateBigHuntRewardConfig(file); err != nil {
		t.Fatal(err)
	}
}

func TestBigHuntRewardUpdatesRejectInvalidValues(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	file, err := memorydb.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = BuildUpdate(path, UpdateRequest{
		ExpectedVersion: file.Version(),
		Changes: []Change{{
			Table: bigHuntRewardGroupTable, Row: 0, Field: "Count", Value: "0",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "must be greater than zero") {
		t.Fatalf("invalid reward count error = %v", err)
	}

	_, _, err = BuildUpdate(path, UpdateRequest{
		ExpectedVersion: file.Version(),
		Changes: []Change{{
			Table: bigHuntBossQuestTable, Row: 0, Field: "BigHuntScoreRewardGroupScheduleId", Value: "2147483647",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown BigHuntScoreRewardGroupScheduleId") {
		t.Fatalf("invalid reward schedule reference error = %v", err)
	}
}
