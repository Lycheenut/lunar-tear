package memorydb

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"

	"github.com/pierrec/lz4/v4"
	"github.com/vmihailenco/msgpack/v5"
)

// File is an editable, isolated master-data snapshot. It never changes the
// package-global database used by the running game server.
type File struct {
	encrypted []byte
	toc       map[string][2]int
	dataBlob  []byte
}

// Int64Edit identifies one int64 cell in a MessagePack table.
type Int64Edit struct {
	Table  string
	Row    int
	Column int
	Value  int64
}

// CellEdit identifies one editable scalar cell in a MessagePack table.
// Supported values are int32, int64, bool, and string.
type CellEdit struct {
	Table        string
	Row          int
	Column       int
	Value        interface{}
	requireInt64 bool
}

// OpenFile reads and decodes a master-data file without publishing it.
func OpenFile(path string) (*File, error) {
	encrypted, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read master data: %w", err)
	}
	return OpenBytes(encrypted)
}

// OpenBytes decodes an encrypted master-data snapshot.
func OpenBytes(encrypted []byte) (*File, error) {
	decrypted, err := decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	toc, dataBlob, err := parseHeader(decrypted)
	if err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	for name, offLen := range toc {
		if offLen[0] < 0 || offLen[1] < 0 || offLen[0]+offLen[1] > len(dataBlob) {
			return nil, fmt.Errorf("table %q points outside the data blob", name)
		}
	}
	return &File{
		encrypted: append([]byte(nil), encrypted...),
		toc:       toc,
		dataBlob:  dataBlob,
	}, nil
}

// Version returns a stable optimistic-lock token for this snapshot.
func (f *File) Version() string {
	return fmt.Sprintf("%x", sha256.Sum256(f.encrypted))
}

// TableRows decodes one table into row/column values for administrative
// inspection. The returned bool is false when the table is absent.
func (f *File) TableRows(name string) ([][]interface{}, bool, error) {
	offLen, ok := f.toc[name]
	if !ok {
		return nil, false, nil
	}
	table, _, err := decodeTableBlob(f.dataBlob[offLen[0] : offLen[0]+offLen[1]])
	if err != nil {
		return nil, true, fmt.Errorf("decode table %q: %w", name, err)
	}
	dec := msgpack.NewDecoder(bytes.NewReader(table))
	dec.UseLooseInterfaceDecoding(true)
	var rows [][]interface{}
	if err := dec.Decode(&rows); err != nil {
		return nil, true, fmt.Errorf("unmarshal table %q: %w", name, err)
	}
	return rows, true, nil
}

// Rebuild applies int64 edits while preserving the original MessagePack cell
// widths, rebuilds the table-of-contents, and encrypts a new master-data file.
func (f *File) Rebuild(edits []Int64Edit) ([]byte, error) {
	cells := make([]CellEdit, 0, len(edits))
	for _, edit := range edits {
		cells = append(cells, CellEdit{
			Table: edit.Table, Row: edit.Row, Column: edit.Column, Value: edit.Value, requireInt64: true,
		})
	}
	return f.RebuildCells(cells)
}

// RebuildCells applies scalar edits while preserving every untouched
// MessagePack value byte-for-byte, rebuilds the table-of-contents, and
// encrypts a new master-data file.
func (f *File) RebuildCells(edits []CellEdit) ([]byte, error) {
	grouped := make(map[string][]CellEdit)
	seen := make(map[[3]interface{}]struct{}, len(edits))
	for _, edit := range edits {
		key := [3]interface{}{edit.Table, edit.Row, edit.Column}
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate edit for %s row %d column %d", edit.Table, edit.Row, edit.Column)
		}
		seen[key] = struct{}{}
		grouped[edit.Table] = append(grouped[edit.Table], edit)
	}

	replacements := make(map[string][]byte, len(grouped))
	for name, tableEdits := range grouped {
		offLen, ok := f.toc[name]
		if !ok {
			return nil, fmt.Errorf("table %q not found", name)
		}
		raw := f.dataBlob[offLen[0] : offLen[0]+offLen[1]]
		table, compressed, err := decodeTableBlob(raw)
		if err != nil {
			return nil, fmt.Errorf("decode table %q: %w", name, err)
		}
		patched, err := patchCells(table, tableEdits)
		if err != nil {
			return nil, fmt.Errorf("patch table %q: %w", name, err)
		}
		if compressed {
			replacements[name], err = encodeCompressedTable(patched)
			if err != nil {
				return nil, fmt.Errorf("compress table %q: %w", name, err)
			}
		} else {
			replacements[name] = patched
		}
	}

	type tableEntry struct {
		name   string
		offset int
		length int
	}
	ordered := make([]tableEntry, 0, len(f.toc))
	for name, offLen := range f.toc {
		ordered = append(ordered, tableEntry{name: name, offset: offLen[0], length: offLen[1]})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].offset < ordered[j].offset })

	var data bytes.Buffer
	for i, entry := range ordered {
		part, replaced := replacements[entry.name]
		if !replaced {
			part = f.dataBlob[entry.offset : entry.offset+entry.length]
		}
		ordered[i].offset = data.Len()
		ordered[i].length = len(part)
		_, _ = data.Write(part)
	}

	// Preserve the official table order as well as the signed int32 widths. The
	// client walks the TOC in serialized order while consuming the data blob;
	// emitting a Go map randomizes that order even though the explicit offsets
	// still make the file readable by our looser server parser.
	var header bytes.Buffer
	enc := msgpack.NewEncoder(&header)
	if err := enc.EncodeMapLen(len(ordered)); err != nil {
		return nil, fmt.Errorf("encode header: %w", err)
	}
	for _, entry := range ordered {
		if err := enc.EncodeString(entry.name); err != nil {
			return nil, fmt.Errorf("encode header table name %q: %w", entry.name, err)
		}
		if err := enc.EncodeArrayLen(2); err != nil {
			return nil, fmt.Errorf("encode header range for %q: %w", entry.name, err)
		}
		if err := enc.EncodeInt32(int32(entry.offset)); err != nil {
			return nil, fmt.Errorf("encode header offset for %q: %w", entry.name, err)
		}
		if err := enc.EncodeInt32(int32(entry.length)); err != nil {
			return nil, fmt.Errorf("encode header length for %q: %w", entry.name, err)
		}
	}
	decrypted := append(header.Bytes(), data.Bytes()...)
	encrypted, err := encrypt(decrypted)
	if err != nil {
		return nil, err
	}
	if _, err := OpenBytes(encrypted); err != nil {
		return nil, fmt.Errorf("verify rebuilt master data: %w", err)
	}
	return encrypted, nil
}

func decodeTableBlob(raw []byte) ([]byte, bool, error) {
	if len(raw) == 0 {
		return nil, false, fmt.Errorf("empty table blob")
	}
	dec := msgpack.NewDecoder(bytes.NewReader(raw))
	code, extData, err := decodeExt(dec)
	if err != nil || code != lz4ExtCode {
		return append([]byte(nil), raw...), false, nil
	}
	uncompressedSize, lz4Data, err := readLZ4ExtHeader(extData)
	if err != nil {
		return nil, true, err
	}
	decompressed := make([]byte, uncompressedSize)
	n, err := lz4.UncompressBlock(lz4Data, decompressed)
	if err != nil {
		return nil, true, err
	}
	if n != uncompressedSize {
		return nil, true, fmt.Errorf("decompressed %d bytes, expected %d", n, uncompressedSize)
	}
	return decompressed, true, nil
}

func encodeCompressedTable(table []byte) ([]byte, error) {
	compressed := make([]byte, lz4.CompressBlockBound(len(table)))
	n, err := lz4.CompressBlock(table, compressed, nil)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("LZ4 did not produce a block")
	}
	compressed = compressed[:n]

	extData := make([]byte, 5+len(compressed))
	extData[0] = 0xd2
	binary.BigEndian.PutUint32(extData[1:5], uint32(len(table)))
	copy(extData[5:], compressed)

	var result []byte
	switch {
	case len(extData) <= 0xff:
		result = []byte{0xc7, byte(len(extData)), byte(lz4ExtCode)}
	case len(extData) <= 0xffff:
		result = make([]byte, 4)
		result[0] = 0xc8
		binary.BigEndian.PutUint16(result[1:3], uint16(len(extData)))
		result[3] = byte(lz4ExtCode)
	default:
		result = make([]byte, 6)
		result[0] = 0xc9
		binary.BigEndian.PutUint32(result[1:5], uint32(len(extData)))
		result[5] = byte(lz4ExtCode)
	}
	return append(result, extData...), nil
}

func patchInt64Cells(table []byte, edits []Int64Edit) error {
	targets := make(map[[2]int]int64, len(edits))
	for _, edit := range edits {
		if edit.Row < 0 || edit.Column < 0 {
			return fmt.Errorf("negative row or column")
		}
		targets[[2]int{edit.Row, edit.Column}] = edit.Value
	}

	rowCount, pos, err := readArrayLen(table, 0)
	if err != nil {
		return err
	}
	remaining := len(targets)
	for row := 0; row < rowCount; row++ {
		columnCount, next, err := readArrayLen(table, pos)
		if err != nil {
			return fmt.Errorf("row %d: %w", row, err)
		}
		pos = next
		for column := 0; column < columnCount; column++ {
			valuePos := pos
			pos, err = skipMsgpackValue(table, pos)
			if err != nil {
				return fmt.Errorf("row %d column %d: %w", row, column, err)
			}
			value, ok := targets[[2]int{row, column}]
			if !ok {
				continue
			}
			if table[valuePos] != 0xd3 {
				return fmt.Errorf("row %d column %d is not encoded as int64", row, column)
			}
			binary.BigEndian.PutUint64(table[valuePos+1:valuePos+9], uint64(value))
			delete(targets, [2]int{row, column})
			remaining--
		}
	}
	if remaining != 0 {
		return fmt.Errorf("%d edit targets were outside the table", remaining)
	}
	return nil
}

type cellPatch struct {
	start int
	end   int
	value []byte
}

func patchCells(table []byte, edits []CellEdit) ([]byte, error) {
	targets := make(map[[2]int]CellEdit, len(edits))
	for _, edit := range edits {
		if edit.Row < 0 || edit.Column < 0 {
			return nil, fmt.Errorf("negative row or column")
		}
		targets[[2]int{edit.Row, edit.Column}] = edit
	}

	rowCount, pos, err := readArrayLen(table, 0)
	if err != nil {
		return nil, err
	}
	patches := make([]cellPatch, 0, len(targets))
	for row := 0; row < rowCount; row++ {
		columnCount, next, err := readArrayLen(table, pos)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", row, err)
		}
		pos = next
		for column := 0; column < columnCount; column++ {
			valuePos := pos
			pos, err = skipMsgpackValue(table, pos)
			if err != nil {
				return nil, fmt.Errorf("row %d column %d: %w", row, column, err)
			}
			edit, ok := targets[[2]int{row, column}]
			if !ok {
				continue
			}
			encoded, err := encodeCellValue(table[valuePos], edit.Value, edit.requireInt64)
			if err != nil {
				return nil, fmt.Errorf("row %d column %d: %w", row, column, err)
			}
			patches = append(patches, cellPatch{start: valuePos, end: pos, value: encoded})
			delete(targets, [2]int{row, column})
		}
	}
	if len(targets) != 0 {
		return nil, fmt.Errorf("%d edit targets were outside the table", len(targets))
	}
	if len(patches) == 0 {
		return append([]byte(nil), table...), nil
	}

	sort.Slice(patches, func(i, j int) bool { return patches[i].start < patches[j].start })
	var patched bytes.Buffer
	cursor := 0
	for _, patch := range patches {
		_, _ = patched.Write(table[cursor:patch.start])
		_, _ = patched.Write(patch.value)
		cursor = patch.end
	}
	_, _ = patched.Write(table[cursor:])
	return patched.Bytes(), nil
}

func encodeCellValue(originalTag byte, value interface{}, requireInt64 bool) ([]byte, error) {
	switch value := value.(type) {
	case int32:
		if !isIntegerTag(originalTag) {
			return nil, fmt.Errorf("cell is not encoded as integer")
		}
		encoded := make([]byte, 5)
		encoded[0] = 0xd2
		binary.BigEndian.PutUint32(encoded[1:], uint32(value))
		return encoded, nil
	case int64:
		if requireInt64 && originalTag != 0xd3 {
			return nil, fmt.Errorf("cell is not encoded as int64")
		}
		if !requireInt64 && !isIntegerTag(originalTag) {
			return nil, fmt.Errorf("cell is not encoded as integer")
		}
		encoded := make([]byte, 9)
		encoded[0] = 0xd3
		binary.BigEndian.PutUint64(encoded[1:], uint64(value))
		return encoded, nil
	case bool:
		if originalTag != 0xc2 && originalTag != 0xc3 {
			return nil, fmt.Errorf("cell is not encoded as bool")
		}
		if value {
			return []byte{0xc3}, nil
		}
		return []byte{0xc2}, nil
	case string:
		if !isStringTag(originalTag) {
			return nil, fmt.Errorf("cell is not encoded as string")
		}
		encoded, err := msgpack.Marshal(value)
		if err != nil {
			return nil, err
		}
		return encoded, nil
	default:
		return nil, fmt.Errorf("unsupported edit value type %T", value)
	}
}

func isStringTag(tag byte) bool {
	return tag >= 0xa0 && tag <= 0xbf || tag == 0xd9 || tag == 0xda || tag == 0xdb
}

func isIntegerTag(tag byte) bool {
	return tag <= 0x7f || tag >= 0xe0 || tag >= 0xcc && tag <= 0xd3
}

func readArrayLen(data []byte, pos int) (int, int, error) {
	if pos >= len(data) {
		return 0, pos, fmt.Errorf("unexpected end of MessagePack data")
	}
	tag := data[pos]
	switch {
	case tag >= 0x90 && tag <= 0x9f:
		return int(tag & 0x0f), pos + 1, nil
	case tag == 0xdc:
		if pos+3 > len(data) {
			return 0, pos, fmt.Errorf("truncated array16 header")
		}
		return int(binary.BigEndian.Uint16(data[pos+1 : pos+3])), pos + 3, nil
	case tag == 0xdd:
		if pos+5 > len(data) {
			return 0, pos, fmt.Errorf("truncated array32 header")
		}
		length := binary.BigEndian.Uint32(data[pos+1 : pos+5])
		if uint64(length) > uint64(len(data)) {
			return 0, pos, fmt.Errorf("array length is too large")
		}
		return int(length), pos + 5, nil
	default:
		return 0, pos, fmt.Errorf("expected array, found tag 0x%02x", tag)
	}
}

func skipMsgpackValue(data []byte, pos int) (int, error) {
	if pos >= len(data) {
		return pos, fmt.Errorf("unexpected end of MessagePack data")
	}
	tag := data[pos]
	if tag <= 0x7f || tag >= 0xe0 {
		return pos + 1, nil
	}
	if tag >= 0xa0 && tag <= 0xbf {
		return checkedEnd(data, pos, 1+int(tag&0x1f))
	}
	if tag >= 0x90 && tag <= 0x9f {
		return skipMsgpackItems(data, pos+1, int(tag&0x0f))
	}
	if tag >= 0x80 && tag <= 0x8f {
		return skipMsgpackItems(data, pos+1, int(tag&0x0f)*2)
	}

	switch tag {
	case 0xc0, 0xc2, 0xc3:
		return checkedEnd(data, pos, 1)
	case 0xcc, 0xd0:
		return checkedEnd(data, pos, 2)
	case 0xcd, 0xd1, 0xd4:
		return checkedEnd(data, pos, 3)
	case 0xd5:
		return checkedEnd(data, pos, 4)
	case 0xca, 0xce, 0xd2:
		return checkedEnd(data, pos, 5)
	case 0xd6:
		return checkedEnd(data, pos, 6)
	case 0xcb, 0xcf, 0xd3:
		return checkedEnd(data, pos, 9)
	case 0xd7:
		return checkedEnd(data, pos, 10)
	case 0xd8:
		return checkedEnd(data, pos, 18)
	}

	switch tag {
	case 0xc4, 0xd9:
		if pos+2 > len(data) {
			return pos, fmt.Errorf("truncated length header")
		}
		return checkedEnd(data, pos, 2+int(data[pos+1]))
	case 0xc5, 0xda:
		if pos+3 > len(data) {
			return pos, fmt.Errorf("truncated length header")
		}
		return checkedEnd(data, pos, 3+int(binary.BigEndian.Uint16(data[pos+1:pos+3])))
	case 0xc6, 0xdb:
		if pos+5 > len(data) {
			return pos, fmt.Errorf("truncated length header")
		}
		return checkedEnd64(data, pos, 5+uint64(binary.BigEndian.Uint32(data[pos+1:pos+5])))
	case 0xc7:
		if pos+3 > len(data) {
			return pos, fmt.Errorf("truncated ext8 header")
		}
		return checkedEnd(data, pos, 3+int(data[pos+1]))
	case 0xc8:
		if pos+4 > len(data) {
			return pos, fmt.Errorf("truncated ext16 header")
		}
		return checkedEnd(data, pos, 4+int(binary.BigEndian.Uint16(data[pos+1:pos+3])))
	case 0xc9:
		if pos+6 > len(data) {
			return pos, fmt.Errorf("truncated ext32 header")
		}
		return checkedEnd64(data, pos, 6+uint64(binary.BigEndian.Uint32(data[pos+1:pos+5])))
	case 0xdc, 0xdd:
		count, next, err := readArrayLen(data, pos)
		if err != nil {
			return pos, err
		}
		return skipMsgpackItems(data, next, count)
	case 0xde:
		if pos+3 > len(data) {
			return pos, fmt.Errorf("truncated map16 header")
		}
		return skipMsgpackItems(data, pos+3, int(binary.BigEndian.Uint16(data[pos+1:pos+3]))*2)
	case 0xdf:
		if pos+5 > len(data) {
			return pos, fmt.Errorf("truncated map32 header")
		}
		count := uint64(binary.BigEndian.Uint32(data[pos+1:pos+5])) * 2
		if count > uint64(len(data)) {
			return pos, fmt.Errorf("map length is too large")
		}
		return skipMsgpackItems(data, pos+5, int(count))
	default:
		return pos, fmt.Errorf("unsupported MessagePack tag 0x%02x", tag)
	}
}

func skipMsgpackItems(data []byte, pos, count int) (int, error) {
	var err error
	for i := 0; i < count; i++ {
		pos, err = skipMsgpackValue(data, pos)
		if err != nil {
			return pos, err
		}
	}
	return pos, nil
}

func checkedEnd(data []byte, pos, size int) (int, error) {
	return checkedEnd64(data, pos, uint64(size))
}

func checkedEnd64(data []byte, pos int, size uint64) (int, error) {
	end := uint64(pos) + size
	if end > uint64(len(data)) {
		return pos, fmt.Errorf("truncated MessagePack value")
	}
	return int(end), nil
}

func encrypt(data []byte) ([]byte, error) {
	key, err := hex.DecodeString(aesKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	iv, err := hex.DecodeString(aesIVHex)
	if err != nil {
		return nil, fmt.Errorf("decode IV: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	padding := aes.BlockSize - len(data)%aes.BlockSize
	padded := make([]byte, len(data)+padding)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padding)
	}
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(padded, padded)
	return padded, nil
}
