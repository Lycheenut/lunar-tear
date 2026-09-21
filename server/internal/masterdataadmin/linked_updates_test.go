package masterdataadmin

import (
	"errors"
	"lunar-tear/server/internal/masterdata/memorydb"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestIndividualScheduleEditsNeverCascade(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	for _, tableName := range []string{"m_event_quest_chapter", "m_mom_banner", "m_navi_cut_in", "m_shop", "m_consumable_item_term", "m_mission_term", "m_gacha_medal", "m_tip", "m_login_bonus"} {
		t.Run(tableName, func(t *testing.T) {
			table := catalogTableByName(t, catalog, tableName)
			row := table.Rows[0]
			field := table.TimeFields[len(table.TimeFields)-1]
			request := UpdateRequest{ExpectedVersion: catalog.Version, Changes: []Change{{Table: tableName, Row: row.Index, Field: field, Value: row.Times[field] - 1000}}}
			preview, err := PreviewUpdate(path, request)
			if err != nil {
				t.Fatal(err)
			}
			if preview.GeneratedChanges != 0 || len(preview.Impacts) != 0 || preview.TotalChanges != 1 {
				t.Fatalf("unexpected cascade: %+v", preview)
			}
			_, result, err := BuildUpdate(path, request)
			if err != nil {
				t.Fatal(err)
			}
			if result.ChangedRows != 1 || result.ChangedCells != 1 {
				t.Fatalf("unexpected cascade: %+v", result)
			}
		})
	}
}

func linkedUpdateTestCatalog(t *testing.T) (string, *Catalog) {
	t.Helper()
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
	return path, catalog
}

func catalogRowByID(t *testing.T, catalog *Catalog, tableName, fieldName, value string) Row {
	t.Helper()
	for _, table := range catalog.Tables {
		if table.Name != tableName {
			continue
		}
		for _, row := range table.Rows {
			if row.Values[fieldName] == value {
				return row
			}
		}
	}
	t.Fatalf("%s %s=%s not found", tableName, fieldName, value)
	return Row{}
}

func catalogTableByName(t *testing.T, catalog *Catalog, tableName string) Table {
	t.Helper()
	for _, table := range catalog.Tables {
		if table.Name == tableName {
			return table
		}
	}
	t.Fatalf("table %s not found", tableName)
	return Table{}
}

func mustParseInt64(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func assertRawTimeByID(t *testing.T, file *memorydb.File, table string, idColumn int, id int64, timeColumn int, want int64) {
	t.Helper()
	rows, exists, err := file.TableRows(table)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("table %s is absent", table)
	}
	for _, row := range rows {
		rowID, _ := integerAt(row, idColumn)
		if rowID != id {
			continue
		}
		got, err := valueAsInt64(row[timeColumn])
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s id %d time = %d, want %d", table, id, got, want)
		}
		return
	}
	t.Fatalf("%s id %d not found", table, id)
}
