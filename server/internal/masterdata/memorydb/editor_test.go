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

func TestFileRebuildScalarCells(t *testing.T) {
	table, err := msgpack.Marshal([][]interface{}{
		{int32(7), int64(1000), false, "short"},
		{int32(8), int64(2000), true, "untouched"},
	})
	if err != nil {
		t.Fatal(err)
	}
	file, err := OpenBytes(buildEditorTestFile(t, map[string][]byte{"m_test": table}))
	if err != nil {
		t.Fatal(err)
	}

	candidate, err := file.RebuildCells([]CellEdit{
		{Table: "m_test", Row: 0, Column: 0, Value: int32(-9)},
		{Table: "m_test", Row: 0, Column: 1, Value: int64(9000)},
		{Table: "m_test", Row: 0, Column: 2, Value: true},
		{Table: "m_test", Row: 0, Column: 3, Value: "a replacement string that changes the encoded width"},
	})
	if err != nil {
		t.Fatal(err)
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
	if got := testInt64(t, rows[0][0]); got != -9 {
		t.Fatalf("int32 value = %d, want -9", got)
	}
	if got := testInt64(t, rows[0][1]); got != 9000 {
		t.Fatalf("int64 value = %d, want 9000", got)
	}
	if got := rows[0][2]; got != true {
		t.Fatalf("bool value = %#v, want true", got)
	}
	if got := rows[0][3]; got != "a replacement string that changes the encoded width" {
		t.Fatalf("string value = %#v", got)
	}
	if got := rows[1][3]; got != "untouched" {
		t.Fatalf("untouched row changed to %#v", got)
	}
}

func TestFileRebuildScalarCellsRejectsStorageTypeMismatch(t *testing.T) {
	table, err := msgpack.Marshal([][]interface{}{{int32(7), "value"}})
	if err != nil {
		t.Fatal(err)
	}
	file, err := OpenBytes(buildEditorTestFile(t, map[string][]byte{"m_test": table}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.RebuildCells([]CellEdit{{Table: "m_test", Row: 0, Column: 0, Value: "wrong"}}); err == nil {
		t.Fatal("expected storage type mismatch to fail")
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

func TestFileRebuildEncodesTOCValuesAsInt32(t *testing.T) {
	table, err := msgpack.Marshal([][]interface{}{{int64(1)}})
	if err != nil {
		t.Fatal(err)
	}
	file, err := OpenBytes(buildEditorTestFile(t, map[string][]byte{"m_test": table}))
	if err != nil {
		t.Fatal(err)
	}

	rebuilt, err := file.Rebuild(nil)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := decrypt(rebuilt)
	if err != nil {
		t.Fatal(err)
	}
	// map(1), "m_test", array(2), int32(0), int32(table length)
	wantPrefix := []byte{0x81, 0xa6, 'm', '_', 't', 'e', 's', 't', 0x92, 0xd2, 0, 0, 0, 0, 0xd2}
	if !bytes.HasPrefix(decrypted, wantPrefix) {
		t.Fatalf("rebuilt header prefix = %x, want prefix %x", decrypted[:len(wantPrefix)], wantPrefix)
	}
}

func TestFileRebuildOrdersTOCByDataOffset(t *testing.T) {
	tableA, err := msgpack.Marshal([][]interface{}{{int64(1)}})
	if err != nil {
		t.Fatal(err)
	}
	tableB, err := msgpack.Marshal([][]interface{}{{int64(2)}})
	if err != nil {
		t.Fatal(err)
	}
	file, err := OpenBytes(buildEditorTestFile(t, map[string][]byte{
		"m_z_table": tableA,
		"m_a_table": tableB,
	}))
	if err != nil {
		t.Fatal(err)
	}

	rebuilt, err := file.Rebuild(nil)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := decrypt(rebuilt)
	if err != nil {
		t.Fatal(err)
	}
	decoder := msgpack.NewDecoder(bytes.NewReader(decrypted))
	tableCount, err := decoder.DecodeMapLen()
	if err != nil {
		t.Fatal(err)
	}
	previousEnd := 0
	for i := 0; i < tableCount; i++ {
		name, err := decoder.DecodeString()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decoder.DecodeArrayLen(); err != nil {
			t.Fatal(err)
		}
		offset, err := decoder.DecodeInt64()
		if err != nil {
			t.Fatal(err)
		}
		length, err := decoder.DecodeInt64()
		if err != nil {
			t.Fatal(err)
		}
		if int(offset) != previousEnd {
			t.Fatalf("table %q offset = %d, want contiguous offset %d", name, offset, previousEnd)
		}
		previousEnd += int(length)
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
