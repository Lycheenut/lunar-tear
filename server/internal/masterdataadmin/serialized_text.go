package masterdataadmin

import (
	"encoding/binary"
	"fmt"
)

func serializedTextAssets(data []byte) ([]string, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("serialized file header is truncated")
	}
	metadataSize := int(binary.BigEndian.Uint32(data[0:4]))
	version := int(binary.BigEndian.Uint32(data[8:12]))
	dataOffset := int64(binary.BigEndian.Uint32(data[12:16]))
	if version < 9 || version >= 22 {
		return nil, fmt.Errorf("unsupported serialized file version %d", version)
	}
	if metadataSize > len(data) || dataOffset < 0 || dataOffset > int64(len(data)) {
		return nil, fmt.Errorf("invalid serialized file offsets")
	}
	reader := serializedReader{data: data, pos: 20, littleEndian: data[16] == 0}
	if _, err := reader.cstring(); err != nil {
		return nil, err
	}
	if _, err := reader.i32(); err != nil {
		return nil, err
	}
	typeTree, err := reader.u8()
	if err != nil {
		return nil, err
	}
	typeCount, err := reader.i32()
	if err != nil || typeCount < 0 || typeCount > 100000 {
		return nil, fmt.Errorf("invalid serialized type count")
	}
	classIDs := make([]int32, typeCount)
	for index := range classIDs {
		classID, err := reader.i32()
		if err != nil {
			return nil, err
		}
		classIDs[index] = classID
		if version >= 16 {
			if _, err = reader.u8(); err != nil {
				return nil, err
			}
		}
		if version >= 17 {
			if _, err = reader.i16(); err != nil {
				return nil, err
			}
		}
		if version >= 13 {
			if (version < 16 && classID < 0) || (version >= 16 && classID == 114) {
				if err = reader.skip(16); err != nil {
					return nil, err
				}
			}
			if err = reader.skip(16); err != nil {
				return nil, err
			}
		}
		if typeTree != 0 {
			nodeCount, err := reader.i32()
			if err != nil || nodeCount < 0 || nodeCount > 1000000 {
				return nil, fmt.Errorf("invalid serialized type-tree node count")
			}
			stringSize, err := reader.i32()
			if err != nil || stringSize < 0 {
				return nil, fmt.Errorf("invalid serialized type-tree string size")
			}
			nodeSize := 24
			if version >= 19 {
				nodeSize = 32
			}
			if err = reader.skip(int(nodeCount)*nodeSize + int(stringSize)); err != nil {
				return nil, err
			}
		}
		if version >= 21 {
			dependencyCount, err := reader.i32()
			if err != nil || dependencyCount < 0 || dependencyCount > 100000 {
				return nil, fmt.Errorf("invalid serialized dependency count")
			}
			if err = reader.skip(int(dependencyCount) * 4); err != nil {
				return nil, err
			}
		}
	}
	if version >= 7 && version < 14 {
		if _, err = reader.i32(); err != nil {
			return nil, err
		}
	}
	objectCount, err := reader.i32()
	if err != nil || objectCount < 0 || objectCount > 1000000 {
		return nil, fmt.Errorf("invalid serialized object count")
	}
	type textObject struct {
		offset int64
		size   uint32
	}
	var objects []textObject
	for range objectCount {
		reader.align(4)
		if version < 14 {
			if _, err = reader.i32(); err != nil {
				return nil, err
			}
		} else if _, err = reader.i64(); err != nil {
			return nil, err
		}
		start, err := reader.u32()
		if err != nil {
			return nil, err
		}
		size, err := reader.u32()
		if err != nil {
			return nil, err
		}
		typeID, err := reader.i32()
		if err != nil {
			return nil, err
		}
		if typeID >= 0 && int(typeID) < len(classIDs) && classIDs[typeID] == 49 {
			objects = append(objects, textObject{offset: dataOffset + int64(start), size: size})
		}
		if version < 16 {
			if _, err = reader.i16(); err != nil {
				return nil, err
			}
		}
		if version < 11 {
			if _, err = reader.u16(); err != nil {
				return nil, err
			}
		}
		if version >= 11 && version < 17 {
			if _, err = reader.i16(); err != nil {
				return nil, err
			}
		}
		if version == 15 || version == 16 {
			if _, err = reader.u8(); err != nil {
				return nil, err
			}
		}
	}

	texts := make([]string, 0, len(objects))
	for _, object := range objects {
		end := object.offset + int64(object.size)
		if object.offset < 0 || end > int64(len(data)) {
			continue
		}
		objectReader := serializedReader{data: data[object.offset:end], littleEndian: reader.littleEndian}
		nameLength, err := objectReader.i32()
		if err != nil || nameLength < 0 || int(nameLength) > len(objectReader.data)-objectReader.pos {
			continue
		}
		if err = objectReader.skip(int(nameLength)); err != nil {
			continue
		}
		objectReader.align(4)
		textLength, err := objectReader.i32()
		if err != nil || textLength < 0 || int(textLength) > len(objectReader.data)-objectReader.pos {
			continue
		}
		texts = append(texts, string(objectReader.data[objectReader.pos:objectReader.pos+int(textLength)]))
	}
	return texts, nil
}

type serializedReader struct {
	data         []byte
	pos          int
	littleEndian bool
}

func (r *serializedReader) take(size int) ([]byte, error) {
	if size < 0 || r.pos < 0 || r.pos+size > len(r.data) {
		return nil, fmt.Errorf("unexpected end of serialized data")
	}
	value := r.data[r.pos : r.pos+size]
	r.pos += size
	return value, nil
}

func (r *serializedReader) skip(size int) error {
	_, err := r.take(size)
	return err
}

func (r *serializedReader) align(size int) {
	r.pos = (r.pos + size - 1) &^ (size - 1)
}

func (r *serializedReader) cstring() (string, error) {
	start := r.pos
	for r.pos < len(r.data) && r.data[r.pos] != 0 {
		r.pos++
	}
	if r.pos >= len(r.data) {
		return "", fmt.Errorf("unterminated serialized string")
	}
	value := string(r.data[start:r.pos])
	r.pos++
	return value, nil
}

func (r *serializedReader) order() binary.ByteOrder {
	if r.littleEndian {
		return binary.LittleEndian
	}
	return binary.BigEndian
}

func (r *serializedReader) u8() (uint8, error) {
	value, err := r.take(1)
	if err != nil {
		return 0, err
	}
	return value[0], nil
}

func (r *serializedReader) u16() (uint16, error) {
	value, err := r.take(2)
	if err != nil {
		return 0, err
	}
	return r.order().Uint16(value), nil
}

func (r *serializedReader) i16() (int16, error) {
	value, err := r.u16()
	return int16(value), err
}

func (r *serializedReader) u32() (uint32, error) {
	value, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return r.order().Uint32(value), nil
}

func (r *serializedReader) i32() (int32, error) {
	value, err := r.u32()
	return int32(value), err
}

func (r *serializedReader) i64() (int64, error) {
	value, err := r.take(8)
	if err != nil {
		return 0, err
	}
	return int64(r.order().Uint64(value)), nil
}
