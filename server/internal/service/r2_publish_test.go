package service

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildR2PublishPlan(t *testing.T) {
	resetAssetCaches()
	baseDir := t.TempDir()
	writePublishFixture(t, baseDir, "android", "object1", bytesOf('a', 300))

	var progress []R2PublishProgress
	result, err := buildR2PublishPlan(R2PublishOptions{
		BaseDir:          baseDir,
		Revision:         "0",
		ResourceVersion:  "300116832",
		ResourcesBaseURL: "https://assets.example.com",
		DryRun:           true,
		Workers:          2,
		OnProgress: func(update R2PublishProgress) {
			progress = append(progress, update)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(result.Entries))
	}
	entry := result.Entries[0]
	if !strings.HasSuffix(entry.Key, "/resource-bundle-server/unso-300116832-assetbundle/object1") {
		t.Fatalf("unexpected key %q", entry.Key)
	}
	if entry.Platform != "android" {
		t.Fatalf("platform = %q, want android", entry.Platform)
	}
	if len(progress) == 0 {
		t.Fatal("no validation progress reported")
	}
	lastProgress := progress[len(progress)-1]
	if lastProgress.Phase != R2PublishPhaseValidate ||
		lastProgress.Completed != 1 ||
		lastProgress.Total != 1 ||
		lastProgress.BytesProcessed != 300 {
		t.Fatalf("last progress = %+v", lastProgress)
	}
}

func TestBuildR2PublishPlanRejectsPlatformCollision(t *testing.T) {
	resetAssetCaches()
	baseDir := t.TempDir()
	writePublishFixture(t, baseDir, "android", "shared-object", bytesOf('a', 300))
	writePublishFixture(t, baseDir, "ios", "shared-object", bytesOf('b', 300))

	_, err := buildR2PublishPlan(R2PublishOptions{
		BaseDir:          baseDir,
		Revision:         "0",
		ResourceVersion:  "300116832",
		ResourcesBaseURL: "https://assets.example.com",
		DryRun:           true,
	})
	if err == nil || !strings.Contains(err.Error(), "platform collisions") {
		t.Fatalf("error = %v, want platform collision", err)
	}
}

func TestPrepareR2PublishReportsMaterializationProgress(t *testing.T) {
	resetAssetCaches()
	baseDir := t.TempDir()
	writePublishFixture(t, baseDir, "android", "android-object", bytesOf('a', 300))
	writePublishFixture(t, baseDir, "ios", "ios-object", bytesOf('b', 400))
	outputDir := filepath.Join(t.TempDir(), "publish")

	var progress []R2PublishProgress
	result, err := PrepareR2Publish(R2PublishOptions{
		BaseDir:          baseDir,
		OutputDir:        outputDir,
		Revision:         "0",
		ResourceVersion:  "300116832",
		ResourcesBaseURL: "https://assets.example.com",
		Workers:          2,
		OnProgress: func(update R2PublishProgress) {
			progress = append(progress, update)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("entry count = %d, want 2", len(result.Entries))
	}
	if len(progress) == 0 {
		t.Fatal("no progress reported")
	}
	lastProgress := progress[len(progress)-1]
	if lastProgress.Phase != R2PublishPhaseMaterialize ||
		lastProgress.Completed != 2 ||
		lastProgress.Total != 2 ||
		lastProgress.BytesProcessed != 700 {
		t.Fatalf("last progress = %+v", lastProgress)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "manifest.json")); err != nil {
		t.Fatalf("manifest: %v", err)
	}
}

func writePublishFixture(t *testing.T, baseDir, platform, objectID string, content []byte) {
	t.Helper()
	revisionDir := filepath.Join(baseDir, "assets", "revisions", "0", platform)
	assetPath := filepath.Join(revisionDir, "assetbundle", "fixture.assetbundle")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, content, 0644); err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(content)
	entry := appendProtoString(nil, 3, "fixture")
	entry = appendProtoVarint(entry, 4, uint64(len(content)))
	entry = appendProtoString(entry, 10, hex.EncodeToString(sum[:]))
	entry = appendProtoString(entry, 11, objectID)
	listBin := appendVarint(nil, 1<<3|2)
	listBin = appendVarint(listBin, uint64(len(entry)))
	listBin = append(listBin, entry...)
	if err := os.WriteFile(filepath.Join(revisionDir, "list.bin"), listBin, 0644); err != nil {
		t.Fatal(err)
	}
}

func appendProtoString(dst []byte, field int, value string) []byte {
	dst = appendVarint(dst, uint64(field<<3|2))
	dst = appendVarint(dst, uint64(len(value)))
	return append(dst, value...)
}

func appendProtoVarint(dst []byte, field int, value uint64) []byte {
	dst = appendVarint(dst, uint64(field<<3))
	return appendVarint(dst, value)
}

func appendVarint(dst []byte, value uint64) []byte {
	for value >= 0x80 {
		dst = append(dst, byte(value)|0x80)
		value >>= 7
	}
	return append(dst, byte(value))
}

func bytesOf(value byte, count int) []byte {
	out := make([]byte, count)
	for i := range out {
		out[i] = value
	}
	return out
}

func resetAssetCaches() {
	listBinCacheMu.Lock()
	listBinCache = make(map[string]listBinIndex)
	listBinInflight = make(map[string]*listBinLoad)
	listBinCacheMu.Unlock()

	infoCacheMu.Lock()
	infoCache = make(map[string]map[string]infoAlias)
	infoInflight = make(map[string]*infoLoad)
	infoCacheMu.Unlock()

	fileMD5CacheMu.Lock()
	fileMD5Cache = make(map[string]fileMD5Entry)
	fileMD5CacheMu.Unlock()
}
