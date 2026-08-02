package runtime_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdataadmin"
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
