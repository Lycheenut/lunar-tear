package assettext

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReadsAssetsAndPatchOverrides(t *testing.T) {
	assets := filepath.Join(t.TempDir(), "assets")
	masterData := filepath.Join(assets, "release", "master.bin.e")
	textRoot := filepath.Join(assets, "revisions", "0", "assetbundle", "text")
	writeBundle := func(path, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, textBundleFixture(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, language := range []string{"en", "ja"} {
		for _, category := range []string{"character", "gacha_title", "possession/weapon", "possession/costume", "possession/material", "possession/consumable_item", "possession/companion"} {
			text := category + ".test:Base"
			if category == "possession/weapon" {
				text = "# comment\r\nweapon.name.wp001060.1:Base: Name\r\n"
			}
			writeBundle(filepath.Join(textRoot, language, filepath.FromSlash(category)+".assetbundle"), text)
		}
	}
	patch := filepath.Join(textRoot, "en", "possession", "weapon", "weapon_001060.assetbundle")
	writeBundle(patch, "weapon.name.wp001060.1:Patched Name")
	// Some installed patches contain no localized strings.
	writeBundle(filepath.Join(textRoot, "ja", "character", "empty.assetbundle"), "")
	first, err := Load(masterData)
	if err != nil {
		t.Fatal(err)
	}
	if first["en"]["weapon.name.wp001060.1"] != "Patched Name" || first["ja"]["weapon.name.wp001060.1"] != "Base: Name" {
		t.Fatalf("incorrect text or patch precedence: %v", first)
	}
	writeBundle(patch, "weapon.name.wp001060.1:Updated Name")
	second, err := Load(masterData)
	if err != nil {
		t.Fatal(err)
	}
	if second["en"]["weapon.name.wp001060.1"] != "Updated Name" || first["en"]["weapon.name.wp001060.1"] != "Patched Name" {
		t.Fatal("a new startup must read assets again without changing the old snapshot")
	}
	if err := os.WriteFile(patch, []byte("corrupt bundle"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(masterData); err == nil || !strings.Contains(err.Error(), patch) {
		t.Fatalf("corrupt asset error = %v, want offending path", err)
	}
	if _, err := Load(filepath.Join(t.TempDir(), "release", "master.bin.e")); err == nil {
		t.Fatal("missing assets must be reported at startup")
	}
}

// A minimal uncompressed UnityFS bundle with one version-17 TextAsset. Real
// encrypted/LZ4 assets are exercised by the Gacha name-coverage integration test.
func textBundleFixture(text string) []byte {
	var object bytes.Buffer
	binary.Write(&object, binary.LittleEndian, int32(4))
	object.WriteString("text")
	binary.Write(&object, binary.LittleEndian, int32(len(text)))
	object.WriteString(text)

	var serialized bytes.Buffer
	serialized.Write(make([]byte, 20))
	serialized.WriteString("2019.4.0f1\x00")
	binary.Write(&serialized, binary.LittleEndian, int32(13)) // Platform.
	serialized.WriteByte(0)                                   // No type tree.
	binary.Write(&serialized, binary.LittleEndian, int32(1))
	binary.Write(&serialized, binary.LittleEndian, int32(49)) // TextAsset class.
	serialized.WriteByte(0)
	binary.Write(&serialized, binary.LittleEndian, int16(-1))
	serialized.Write(make([]byte, 16))
	binary.Write(&serialized, binary.LittleEndian, int32(1)) // Object count.
	for serialized.Len()%4 != 0 {
		serialized.WriteByte(0)
	}
	binary.Write(&serialized, binary.LittleEndian, int64(1)) // Path ID.
	binary.Write(&serialized, binary.LittleEndian, uint32(0))
	binary.Write(&serialized, binary.LittleEndian, uint32(object.Len()))
	binary.Write(&serialized, binary.LittleEndian, int32(0)) // Type ID.
	dataOffset := serialized.Len()
	serialized.Write(object.Bytes())
	data := serialized.Bytes()
	binary.BigEndian.PutUint32(data[0:4], uint32(dataOffset-20))
	binary.BigEndian.PutUint32(data[4:8], uint32(len(data)))
	binary.BigEndian.PutUint32(data[8:12], 17)
	binary.BigEndian.PutUint32(data[12:16], uint32(dataOffset))

	var info bytes.Buffer
	info.Write(make([]byte, 16))
	binary.Write(&info, binary.BigEndian, uint32(1)) // Block count.
	binary.Write(&info, binary.BigEndian, uint32(len(data)))
	binary.Write(&info, binary.BigEndian, uint32(len(data)))
	binary.Write(&info, binary.BigEndian, uint16(0))
	binary.Write(&info, binary.BigEndian, uint32(1)) // Node count.
	binary.Write(&info, binary.BigEndian, int64(0))
	binary.Write(&info, binary.BigEndian, int64(len(data)))
	binary.Write(&info, binary.BigEndian, uint32(0))
	info.WriteString("CAB-test\x00")

	var bundle bytes.Buffer
	bundle.WriteString("UnityFS\x00")
	binary.Write(&bundle, binary.BigEndian, uint32(6))
	bundle.WriteString("5.x.x\x002019.4.0f1\x00")
	sizeOffset := bundle.Len()
	binary.Write(&bundle, binary.BigEndian, uint64(0))
	binary.Write(&bundle, binary.BigEndian, uint32(info.Len()))
	binary.Write(&bundle, binary.BigEndian, uint32(info.Len()))
	binary.Write(&bundle, binary.BigEndian, uint32(0))
	bundle.Write(info.Bytes())
	bundle.Write(data)
	binary.BigEndian.PutUint64(bundle.Bytes()[sizeOffset:], uint64(bundle.Len()))
	return bundle.Bytes()
}
