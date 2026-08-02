package memorydb

import (
	"bytes"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

func TestFileRebuildInt64Cell(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		t.Run(map[bool]string{false: "plain", true: "compressed"}[compressed], func(t *testing.T) {
			table, err := msgpack.Marshal([][]interface{}{
				{int32(7), int64(1000), int64(2000), "keep"},
				{int32(8), int64(3000), int64(4000), "untouched"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if compressed {
				table, err = encodeCompressedTable(table)
				if err != nil {
					t.Fatal(err)
				}
			}
			encrypted := buildEditorTestFile(t, map[string][]byte{"m_test": table})
			file, err := OpenBytes(encrypted)
			if err != nil {
				t.Fatal(err)
			}

			candidate, err := file.Rebuild([]Int64Edit{{Table: "m_test", Row: 0, Column: 2, Value: 9000}})
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(candidate, encrypted) {
				t.Fatal("rebuilt file is byte-identical to its input")
			}
			rebuilt, err := OpenBytes(candidate)
			if err != nil {
				t.Fatal(err)
			}
			rows, exists, err := rebuilt.TableRows("m_test")
			if err != nil {
				t.Fatal(err)
			}
			if !exists || len(rows) != 2 {
				t.Fatalf("unexpected rows: exists=%v len=%d", exists, len(rows))
			}
			if got := testInt64(t, rows[0][2]); got != 9000 {
				t.Fatalf("edited value = %d, want 9000", got)
			}
			if got := testInt64(t, rows[0][1]); got != 1000 {
				t.Fatalf("adjacent value changed to %d", got)
			}
			if got := testInt64(t, rows[1][2]); got != 4000 {
				t.Fatalf("other row changed to %d", got)
			}
			if got := rows[0][3]; got != "keep" {
				t.Fatalf("non-time value changed to %#v", got)
			}
		})
	}
}

func TestFileRebuildRejectsNonInt64Cell(t *testing.T) {
	table, err := msgpack.Marshal([][]interface{}{{int32(7), int64(1000), int64(2000)}})
	if err != nil {
		t.Fatal(err)
	}
	file, err := OpenBytes(buildEditorTestFile(t, map[string][]byte{"m_test": table}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Rebuild([]Int64Edit{{Table: "m_test", Row: 0, Column: 0, Value: 9}}); err == nil {
		t.Fatal("expected editing an int32 cell to fail")
	}
}

func buildEditorTestFile(t *testing.T, tables map[string][]byte) []byte {
	t.Helper()
	toc := make(map[string][2]int, len(tables))
	var data bytes.Buffer
	for name, table := range tables {
		toc[name] = [2]int{data.Len(), len(table)}
		_, _ = data.Write(table)
	}
	var header bytes.Buffer
	encoder := msgpack.NewEncoder(&header)
	encoder.SetSortMapKeys(true)
	if err := encoder.Encode(toc); err != nil {
		t.Fatal(err)
	}
	encrypted, err := encrypt(append(header.Bytes(), data.Bytes()...))
	if err != nil {
		t.Fatal(err)
	}
	return encrypted
}

func testInt64(t *testing.T, value interface{}) int64 {
	t.Helper()
	switch value := value.(type) {
	case int64:
		return value
	case int:
		return int64(value)
	default:
		t.Fatalf("value has type %T", value)
		return 0
	}
}
