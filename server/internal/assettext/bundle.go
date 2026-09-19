// AssetBundle decoding follows the configuration manager on the prod branch.
package assettext

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pierrec/lz4/v4"
)

func readTextAssetBundle(path, bundleRoot string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw, err = unmaskAssetBundle(raw, path, bundleRoot)
	if err != nil {
		return nil, err
	}
	nodes, err := unityFSNodes(raw)
	if err != nil {
		return nil, err
	}
	entries := make(map[string]string)
	for name, node := range nodes {
		if strings.HasSuffix(strings.ToLower(name), ".ress") {
			continue
		}
		assets, err := serializedTextAssets(node)
		if err != nil {
			return nil, fmt.Errorf("read TextAssets from %s: %w", name, err)
		}
		for _, asset := range assets {
			for key, value := range parseLocalizedText(asset) {
				entries[key] = value
			}
		}
	}
	return entries, nil
}

func parseLocalizedText(text string) map[string]string {
	entries := make(map[string]string)
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			entries[key] = value
		}
	}
	return entries
}

func unmaskAssetBundle(raw []byte, path, bundleRoot string) ([]byte, error) {
	if len(raw) >= 7 && string(raw[:7]) == "UnityFS" {
		return raw, nil
	}
	if len(raw) < 2 {
		return nil, fmt.Errorf("empty asset bundle")
	}
	relative, err := filepath.Rel(bundleRoot, path)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSuffix(filepath.ToSlash(relative), ".assetbundle")
	mask := stringToMaskBytes(strings.ReplaceAll(name, "/", ")"))
	if len(mask) == 0 {
		return nil, fmt.Errorf("empty asset mask")
	}
	data := append([]byte(nil), raw[1:]...)
	var unmaskLength int
	switch raw[0] {
	case 0x32:
		unmaskLength = len(data)
	case 0x31:
		// Octo's 256-byte header includes the version byte removed above.
		unmaskLength = min(255, len(data))
	default:
		return nil, fmt.Errorf("unknown asset bundle mask version %#x", raw[0])
	}
	for index := 0; index < unmaskLength; index++ {
		data[index] ^= mask[(index+1)%len(mask)]
	}
	return append([]byte{'U'}, data...), nil
}

func stringToMaskBytes(name string) []byte {
	chars := []byte(name)
	mask := make([]byte, len(chars)*2)
	for index, value := range chars {
		mask[index*2] = value
		mask[len(mask)-1-index*2] = ^value
	}
	hash := byte(0xbb)
	for _, value := range mask {
		hash = (hash&1)<<7 | hash>>1
		hash ^= value
	}
	for index := range mask {
		mask[index] ^= hash
	}
	return mask
}

func unityFSNodes(data []byte) (map[string][]byte, error) {
	reader := byteReader{data: data}
	signature, err := reader.cstring()
	if err != nil || signature != "UnityFS" {
		return nil, fmt.Errorf("invalid UnityFS signature")
	}
	format, err := reader.be32()
	if err != nil {
		return nil, err
	}
	if _, err = reader.cstring(); err != nil {
		return nil, err
	}
	if _, err = reader.cstring(); err != nil {
		return nil, err
	}
	fileSize, err := reader.be64()
	if err != nil {
		return nil, err
	}
	compressedInfoSize, err := reader.be32()
	if err != nil {
		return nil, err
	}
	uncompressedInfoSize, err := reader.be32()
	if err != nil {
		return nil, err
	}
	flags, err := reader.be32()
	if err != nil {
		return nil, err
	}
	if format >= 7 {
		reader.align(16)
	}
	if fileSize > uint64(len(data)) || compressedInfoSize > uint32(len(data)) {
		return nil, fmt.Errorf("invalid UnityFS size")
	}

	infoOffset := reader.pos
	if flags&0x80 != 0 {
		infoOffset = int(fileSize) - int(compressedInfoSize)
	}
	infoEnd := infoOffset + int(compressedInfoSize)
	if infoOffset < 0 || infoEnd > len(data) {
		return nil, fmt.Errorf("UnityFS block info is outside the file")
	}
	info, err := decompressUnityBlock(data[infoOffset:infoEnd], int(uncompressedInfoSize), flags&0x3f)
	if err != nil {
		return nil, fmt.Errorf("decompress UnityFS block info: %w", err)
	}
	blocks, nodeList, err := parseUnityFSInfo(info)
	if err != nil {
		return nil, err
	}

	dataOffset := reader.pos
	if flags&0x80 == 0 {
		dataOffset = infoEnd
	}
	if flags&0x200 != 0 {
		dataOffset = (dataOffset + 15) &^ 15
	}
	var contents []byte
	for _, block := range blocks {
		end := dataOffset + int(block.compressedSize)
		if dataOffset < 0 || end > len(data) {
			return nil, fmt.Errorf("UnityFS data block is outside the file")
		}
		part, err := decompressUnityBlock(data[dataOffset:end], int(block.uncompressedSize), uint32(block.flags)&0x3f)
		if err != nil {
			return nil, fmt.Errorf("decompress UnityFS data block: %w", err)
		}
		contents = append(contents, part...)
		dataOffset = end
	}

	nodes := make(map[string][]byte, len(nodeList))
	for _, node := range nodeList {
		end := node.offset + node.size
		if node.offset < 0 || node.size < 0 || end > int64(len(contents)) {
			return nil, fmt.Errorf("UnityFS node %q is outside the data blocks", node.path)
		}
		nodes[node.path] = contents[node.offset:end]
	}
	return nodes, nil
}

type unityBlock struct {
	uncompressedSize uint32
	compressedSize   uint32
	flags            uint16
}

type unityNode struct {
	offset int64
	size   int64
	path   string
}

func parseUnityFSInfo(data []byte) ([]unityBlock, []unityNode, error) {
	reader := byteReader{data: data, pos: 16}
	blockCount, err := reader.be32()
	if err != nil || blockCount > 100000 {
		return nil, nil, fmt.Errorf("invalid UnityFS block count")
	}
	blocks := make([]unityBlock, 0, blockCount)
	for range blockCount {
		uncompressedSize, err := reader.be32()
		if err != nil {
			return nil, nil, err
		}
		compressedSize, err := reader.be32()
		if err != nil {
			return nil, nil, err
		}
		flags, err := reader.be16()
		if err != nil {
			return nil, nil, err
		}
		blocks = append(blocks, unityBlock{uncompressedSize: uncompressedSize, compressedSize: compressedSize, flags: flags})
	}
	nodeCount, err := reader.be32()
	if err != nil || nodeCount > 100000 {
		return nil, nil, fmt.Errorf("invalid UnityFS node count")
	}
	nodes := make([]unityNode, 0, nodeCount)
	for range nodeCount {
		offset, err := reader.be64()
		if err != nil {
			return nil, nil, err
		}
		size, err := reader.be64()
		if err != nil {
			return nil, nil, err
		}
		if _, err = reader.be32(); err != nil {
			return nil, nil, err
		}
		path, err := reader.cstring()
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, unityNode{offset: int64(offset), size: int64(size), path: path})
	}
	return blocks, nodes, nil
}

func decompressUnityBlock(source []byte, size int, compression uint32) ([]byte, error) {
	switch compression {
	case 0:
		if len(source) != size {
			return nil, fmt.Errorf("uncompressed block has size %d, expected %d", len(source), size)
		}
		return append([]byte(nil), source...), nil
	case 2, 3:
		result := make([]byte, size)
		count, err := lz4.UncompressBlock(source, result)
		if err != nil {
			return nil, err
		}
		if count != size {
			return nil, fmt.Errorf("LZ4 produced %d bytes, expected %d", count, size)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported Unity compression %d", compression)
	}
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) take(size int) ([]byte, error) {
	if size < 0 || r.pos < 0 || r.pos+size > len(r.data) {
		return nil, fmt.Errorf("unexpected end of binary data")
	}
	value := r.data[r.pos : r.pos+size]
	r.pos += size
	return value, nil
}

func (r *byteReader) cstring() (string, error) {
	start := r.pos
	for r.pos < len(r.data) && r.data[r.pos] != 0 {
		r.pos++
	}
	if r.pos >= len(r.data) {
		return "", fmt.Errorf("unterminated string")
	}
	value := string(r.data[start:r.pos])
	r.pos++
	return value, nil
}

func (r *byteReader) align(size int) {
	r.pos = (r.pos + size - 1) &^ (size - 1)
}

func (r *byteReader) be16() (uint16, error) {
	value, err := r.take(2)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(value), nil
}

func (r *byteReader) be32() (uint32, error) {
	value, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(value), nil
}

func (r *byteReader) be64() (uint64, error) {
	value, err := r.take(8)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(value), nil
}
