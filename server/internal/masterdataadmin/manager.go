// Package masterdataadmin exposes the schedule-shaped subset of master data
// and produces validated replacement binaries for the live admin service.
package masterdataadmin

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"lunar-tear/server/internal/masterdata/memorydb"
)

const maxDatetimeMillis int64 = 253402300799999

var ErrVersionConflict = errors.New("master data changed since it was loaded")

type FieldValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Row struct {
	Index         int               `json:"index"`
	Identity      []FieldValue      `json:"identity"`
	Times         map[string]int64  `json:"times"`
	Titles        map[string]string `json:"titles,omitempty"`
	ShopRelations []ShopRelation    `json:"shopRelations,omitempty"`
}

type Table struct {
	Name       string     `json:"name"`
	EntityName string     `json:"entityName"`
	TimeFields []string   `json:"timeFields"`
	Pairs      []timePair `json:"pairs"`
	Rows       []Row      `json:"rows"`
}

type Catalog struct {
	Version         string   `json:"version"`
	DefaultLanguage string   `json:"defaultLanguage"`
	Languages       []string `json:"languages"`
	TableCount      int      `json:"tableCount"`
	RowCount        int      `json:"rowCount"`
	Tables          []Table  `json:"tables"`
}

type Change struct {
	Table string `json:"table"`
	Row   int    `json:"row"`
	Field string `json:"field"`
	Value int64  `json:"value"`
}

type UpdateRequest struct {
	ExpectedVersion string   `json:"expectedVersion"`
	Changes         []Change `json:"changes"`
}

type UpdateResult struct {
	Version      string `json:"version"`
	ChangedCells int    `json:"changedCells"`
	ChangedRows  int    `json:"changedRows"`
}

func Load(path string) (*Catalog, error) {
	file, err := memorydb.OpenFile(path)
	if err != nil {
		return nil, err
	}
	resolver := newTitleResolver(file, loadLocalizationIndex(path))
	return catalogFromFile(file, resolver)
}

func BuildUpdate(path string, request UpdateRequest) ([]byte, UpdateResult, error) {
	if request.ExpectedVersion == "" {
		return nil, UpdateResult{}, fmt.Errorf("expectedVersion is required")
	}
	if len(request.Changes) == 0 {
		return nil, UpdateResult{}, fmt.Errorf("at least one change is required")
	}
	if len(request.Changes) > 10000 {
		return nil, UpdateResult{}, fmt.Errorf("too many changes")
	}

	file, err := memorydb.OpenFile(path)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	if file.Version() != request.ExpectedVersion {
		return nil, UpdateResult{}, ErrVersionConflict
	}

	specByName := make(map[string]tableSpec, len(scheduleTableSpecs))
	rowsByTable := make(map[string][][]interface{})
	for _, spec := range scheduleTableSpecs {
		specByName[spec.Name] = spec
	}
	overrides := make(map[string]map[int]map[string]int64)
	seen := make(map[string]struct{}, len(request.Changes))
	var edits []memorydb.Int64Edit
	changedRows := make(map[string]struct{})

	for _, change := range request.Changes {
		spec, ok := specByName[change.Table]
		if !ok {
			return nil, UpdateResult{}, fmt.Errorf("table %q is not an editable schedule table", change.Table)
		}
		field, ok := findTimeField(spec, change.Field)
		if !ok {
			return nil, UpdateResult{}, fmt.Errorf("field %q is not editable on table %q", change.Field, change.Table)
		}
		if change.Value < 0 || change.Value > maxDatetimeMillis {
			return nil, UpdateResult{}, fmt.Errorf("%s row %d field %s is outside the supported datetime range", change.Table, change.Row, change.Field)
		}
		editKey := fmt.Sprintf("%s\x00%d\x00%s", change.Table, change.Row, change.Field)
		if _, duplicate := seen[editKey]; duplicate {
			return nil, UpdateResult{}, fmt.Errorf("duplicate change for %s row %d field %s", change.Table, change.Row, change.Field)
		}
		seen[editKey] = struct{}{}

		rows, loaded := rowsByTable[change.Table]
		if !loaded {
			var exists bool
			rows, exists, err = file.TableRows(change.Table)
			if err != nil {
				return nil, UpdateResult{}, err
			}
			if !exists {
				return nil, UpdateResult{}, fmt.Errorf("table %q is absent from the current master data", change.Table)
			}
			rowsByTable[change.Table] = rows
		}
		if change.Row < 0 || change.Row >= len(rows) {
			return nil, UpdateResult{}, fmt.Errorf("row %d is outside table %q", change.Row, change.Table)
		}
		if field.Index >= len(rows[change.Row]) {
			return nil, UpdateResult{}, fmt.Errorf("field %q is outside row %d in table %q", change.Field, change.Row, change.Table)
		}
		current, err := valueAsInt64(rows[change.Row][field.Index])
		if err != nil {
			return nil, UpdateResult{}, fmt.Errorf("%s row %d field %s: %w", change.Table, change.Row, change.Field, err)
		}
		if current == change.Value {
			continue
		}
		if overrides[change.Table] == nil {
			overrides[change.Table] = make(map[int]map[string]int64)
		}
		if overrides[change.Table][change.Row] == nil {
			overrides[change.Table][change.Row] = make(map[string]int64)
		}
		overrides[change.Table][change.Row][change.Field] = change.Value
		edits = append(edits, memorydb.Int64Edit{Table: change.Table, Row: change.Row, Column: field.Index, Value: change.Value})
		changedRows[fmt.Sprintf("%s\x00%d", change.Table, change.Row)] = struct{}{}
	}
	if len(edits) == 0 {
		return nil, UpdateResult{}, fmt.Errorf("the submitted values are unchanged")
	}

	for tableName, rowOverrides := range overrides {
		spec := specByName[tableName]
		rows := rowsByTable[tableName]
		for rowIndex, values := range rowOverrides {
			for _, pair := range spec.pairs() {
				start, err := effectiveTime(rows[rowIndex], spec, pair.Start, values)
				if err != nil {
					return nil, UpdateResult{}, fmt.Errorf("%s row %d: %w", tableName, rowIndex, err)
				}
				end, err := effectiveTime(rows[rowIndex], spec, pair.End, values)
				if err != nil {
					return nil, UpdateResult{}, fmt.Errorf("%s row %d: %w", tableName, rowIndex, err)
				}
				// End=0 is used by this master data to explicitly disable a row.
				if end != 0 && start > end {
					return nil, UpdateResult{}, fmt.Errorf("%s must not be after %s", pair.Start, pair.End)
				}
			}
		}
	}

	candidate, err := file.Rebuild(edits)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	candidateFile, err := memorydb.OpenBytes(candidate)
	if err != nil {
		return nil, UpdateResult{}, fmt.Errorf("verify candidate: %w", err)
	}
	return candidate, UpdateResult{
		Version:      candidateFile.Version(),
		ChangedCells: len(edits),
		ChangedRows:  len(changedRows),
	}, nil
}

func catalogFromFile(file *memorydb.File, resolver *titleResolver) (*Catalog, error) {
	catalog := &Catalog{
		Version:         file.Version(),
		DefaultLanguage: "en",
		Languages:       append([]string(nil), supportedLanguages...),
	}
	for _, spec := range scheduleTableSpecs {
		rows, exists, err := file.TableRows(spec.Name)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		table := Table{
			Name:       spec.Name,
			EntityName: spec.EntityName,
			Pairs:      spec.pairs(),
			Rows:       make([]Row, 0, len(rows)),
		}
		for _, field := range spec.Times {
			table.TimeFields = append(table.TimeFields, field.Name)
		}
		for rowIndex, values := range rows {
			row := Row{Index: rowIndex, Times: make(map[string]int64, len(spec.Times))}
			for _, key := range spec.Keys {
				if key.Index >= len(values) {
					return nil, fmt.Errorf("table %q row %d is missing key column %s", spec.Name, rowIndex, key.Name)
				}
				row.Identity = append(row.Identity, FieldValue{Name: key.Name, Value: formatValue(values[key.Index])})
			}
			for _, field := range spec.Times {
				if field.Index >= len(values) {
					return nil, fmt.Errorf("table %q row %d is missing time column %s", spec.Name, rowIndex, field.Name)
				}
				value, err := valueAsInt64(values[field.Index])
				if err != nil {
					return nil, fmt.Errorf("table %q row %d field %s: %w", spec.Name, rowIndex, field.Name, err)
				}
				row.Times[field.Name] = value
			}
			row.Titles = resolver.resolve(spec.Name, values)
			row.ShopRelations = resolver.resolveShopRelations(spec.Name, values)
			table.Rows = append(table.Rows, row)
		}
		catalog.RowCount += len(table.Rows)
		catalog.Tables = append(catalog.Tables, table)
	}
	catalog.TableCount = len(catalog.Tables)
	return catalog, nil
}

func findTimeField(spec tableSpec, name string) (columnSpec, bool) {
	for _, field := range spec.Times {
		if field.Name == name {
			return field, true
		}
	}
	return columnSpec{}, false
}

func effectiveTime(row []interface{}, spec tableSpec, fieldName string, overrides map[string]int64) (int64, error) {
	if value, ok := overrides[fieldName]; ok {
		return value, nil
	}
	field, ok := findTimeField(spec, fieldName)
	if !ok || field.Index >= len(row) {
		return 0, fmt.Errorf("missing field %s", fieldName)
	}
	return valueAsInt64(row[field.Index])
}

func valueAsInt64(value interface{}) (int64, error) {
	switch value := value.(type) {
	case int:
		return int64(value), nil
	case int8:
		return int64(value), nil
	case int16:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case int64:
		return value, nil
	case uint:
		if uint64(value) > math.MaxInt64 {
			return 0, fmt.Errorf("integer overflows int64")
		}
		return int64(value), nil
	case uint8:
		return int64(value), nil
	case uint16:
		return int64(value), nil
	case uint32:
		return int64(value), nil
	case uint64:
		if value > math.MaxInt64 {
			return 0, fmt.Errorf("integer overflows int64")
		}
		return int64(value), nil
	default:
		return 0, fmt.Errorf("expected integer, got %T", value)
	}
}

func formatValue(value interface{}) string {
	switch value := value.(type) {
	case string:
		return value
	case []byte:
		return fmt.Sprintf("%x", value)
	case nil:
		return "null"
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
