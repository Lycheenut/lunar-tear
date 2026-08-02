package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestScheduleSpecsCoverPairedTables(t *testing.T) {
	if got, want := len(scheduleTableSpecs), 38; got != want {
		t.Fatalf("schedule spec count = %d, want %d", got, want)
	}
	seen := make(map[string]bool, len(scheduleTableSpecs))
	for _, spec := range scheduleTableSpecs {
		if seen[spec.Name] {
			t.Fatalf("duplicate table spec %q", spec.Name)
		}
		seen[spec.Name] = true
		if len(spec.pairs()) == 0 {
			t.Errorf("table %q has no start/end pair", spec.Name)
		}
		if len(spec.Times) < 2 {
			t.Errorf("table %q exposes fewer than two time fields", spec.Name)
		}
	}
}

func TestBuildUpdateAgainstCurrentMasterData(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.TableCount != len(scheduleTableSpecs) {
		t.Fatalf("loaded %d schedule tables, want %d", catalog.TableCount, len(scheduleTableSpecs))
	}
	if catalog.RowCount == 0 {
		t.Fatal("loaded catalog has no rows")
	}

	var table Table
	for _, candidate := range catalog.Tables {
		if len(candidate.Rows) > 0 && len(candidate.Pairs) > 0 {
			table = candidate
			break
		}
	}
	if table.Name == "" {
		t.Fatal("no editable schedule row found")
	}
	row := table.Rows[0]
	endField := table.Pairs[0].End
	current := row.Times[endField]
	updated := current + 1000
	if current == 0 || updated > maxDatetimeMillis {
		updated = 1
	}
	candidate, result, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes:         []Change{{Table: table.Name, Row: row.Index, Field: endField, Value: updated}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedCells != 1 || result.ChangedRows != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows, exists, err := rebuilt.TableRows(table.Name)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("rebuilt table %q is absent", table.Name)
	}
	spec, ok := findSpec(table.Name)
	if !ok {
		t.Fatalf("spec %q is absent", table.Name)
	}
	field, ok := findTimeField(spec, endField)
	if !ok {
		t.Fatalf("field %q is absent", endField)
	}
	got, err := valueAsInt64(rows[row.Index][field.Index])
	if err != nil {
		t.Fatal(err)
	}
	if got != updated {
		t.Fatalf("rebuilt value = %d, want %d", got, updated)
	}
}

func findSpec(name string) (tableSpec, bool) {
	for _, spec := range scheduleTableSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return tableSpec{}, false
}
