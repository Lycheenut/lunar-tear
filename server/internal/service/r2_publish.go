package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type R2PublishOptions struct {
	BaseDir          string
	OutputDir        string
	Revision         string
	ResourceVersion  string
	ResourcesBaseURL string
	DryRun           bool
}

type R2PublishEntry struct {
	Key      string `json:"key"`
	Platform string `json:"platform"`
	Type     string `json:"type"`
	ObjectID string `json:"object_id"`
	Source   string `json:"source"`
	Size     int64  `json:"size"`
	MD5      string `json:"md5"`
}

type R2PublishResult struct {
	ResourcesBaseURL string           `json:"resources_base_url"`
	Revision         string           `json:"revision"`
	ResourceVersion  string           `json:"resource_version"`
	Entries          []R2PublishEntry `json:"entries"`
}

// PrepareR2Publish resolves list.bin object IDs to local files, rejects
// platform collisions, and materializes the exact object keys requested by the
// client under OutputDir. Files are hard-linked when possible and copied
// otherwise.
func PrepareR2Publish(options R2PublishOptions) (R2PublishResult, error) {
	result, err := buildR2PublishPlan(options)
	if err != nil || options.DryRun {
		return result, err
	}
	if options.OutputDir == "" {
		return result, fmt.Errorf("output directory is required")
	}
	if err := requireEmptyDirectory(options.OutputDir); err != nil {
		return result, err
	}

	for _, entry := range result.Entries {
		target := filepath.Join(options.OutputDir, filepath.FromSlash(entry.Key))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return result, fmt.Errorf("create output directory for %s: %w", entry.Key, err)
		}
		source := filepath.Join(options.BaseDir, filepath.FromSlash(entry.Source))
		if err := linkOrCopy(source, target); err != nil {
			return result, fmt.Errorf("materialize %s: %w", entry.Key, err)
		}
	}

	manifest, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return result, fmt.Errorf("encode manifest: %w", err)
	}
	manifest = append(manifest, '\n')
	if err := os.WriteFile(filepath.Join(options.OutputDir, "manifest.json"), manifest, 0644); err != nil {
		return result, fmt.Errorf("write manifest: %w", err)
	}
	return result, nil
}

func buildR2PublishPlan(options R2PublishOptions) (R2PublishResult, error) {
	if options.BaseDir == "" {
		options.BaseDir = "."
	}
	if options.Revision == "" {
		options.Revision = "0"
	}
	if options.ResourceVersion == "" || strings.ContainsAny(options.ResourceVersion, `/\`) {
		return R2PublishResult{}, fmt.Errorf("resource version is required and must be a single path segment")
	}
	resourcesBaseURL, err := NormalizeResourcesBaseURL(options.ResourcesBaseURL)
	if err != nil {
		return R2PublishResult{}, err
	}
	base, err := url.Parse(resourcesBaseURL)
	if err != nil {
		return R2PublishResult{}, fmt.Errorf("parse normalized resources base URL: %w", err)
	}
	keyPrefix := strings.Trim(base.Path, "/")

	platforms := availablePublishPlatforms(options.BaseDir, options.Revision)
	if len(platforms) == 0 {
		return R2PublishResult{}, fmt.Errorf("no list.bin found for revision %s", options.Revision)
	}

	byKey := make(map[string]R2PublishEntry)
	var missing []string
	var collisions []string
	for _, platform := range platforms {
		index, ok := loadListBinIndex(options.BaseDir, options.Revision, platform)
		if !ok {
			continue
		}
		objectIDs := make([]string, 0, len(index))
		for objectID := range index {
			objectIDs = append(objectIDs, objectID)
		}
		sort.Strings(objectIDs)

		for _, objectID := range objectIDs {
			if !safeObjectID(objectID) {
				missing = append(missing, fmt.Sprintf("%s:%q has an unsafe object ID", publishPlatformName(platform), objectID))
				continue
			}
			found := false
			for _, assetType := range []string{"assetbundle", "resources"} {
				candidates, listSize, ok := objectIdToFilePathCandidates(
					options.BaseDir, options.Revision, platform, assetType, objectID,
				)
				if !ok {
					continue
				}
				source, size, md5sum, ok := selectPublishCandidate(candidates, listSize)
				if !ok {
					continue
				}
				relativeSource, err := filepath.Rel(options.BaseDir, source)
				if err != nil || relativeSource == ".." || strings.HasPrefix(relativeSource, ".."+string(filepath.Separator)) {
					return R2PublishResult{}, fmt.Errorf("asset source %s is outside assets directory %s", source, options.BaseDir)
				}
				found = true
				key := path.Join(
					keyPrefix,
					"resource-bundle-server",
					"unso-"+options.ResourceVersion+"-"+assetType,
					objectID,
				)
				entry := R2PublishEntry{
					Key:      key,
					Platform: publishPlatformName(platform),
					Type:     assetType,
					ObjectID: objectID,
					Source:   filepath.ToSlash(relativeSource),
					Size:     size,
					MD5:      md5sum,
				}
				if previous, exists := byKey[key]; exists {
					if previous.Size != entry.Size || !strings.EqualFold(previous.MD5, entry.MD5) {
						collisions = append(collisions, fmt.Sprintf(
							"%s differs between %s (%s) and %s (%s)",
							key, previous.Platform, previous.MD5, entry.Platform, entry.MD5,
						))
					}
					continue
				}
				byKey[key] = entry
			}
			if !found {
				missing = append(missing, fmt.Sprintf("%s:%q has no valid local asset", publishPlatformName(platform), objectID))
			}
		}
	}
	if len(collisions) > 0 {
		return R2PublishResult{}, fmt.Errorf("platform collisions prevent direct R2 publishing:\n%s", strings.Join(collisions, "\n"))
	}
	if len(missing) > 0 {
		const maxReported = 20
		reported := missing
		if len(reported) > maxReported {
			reported = reported[:maxReported]
			reported = append(reported, fmt.Sprintf("... and %d more", len(missing)-maxReported))
		}
		return R2PublishResult{}, fmt.Errorf("resource publication is incomplete:\n%s", strings.Join(reported, "\n"))
	}

	entries := make([]R2PublishEntry, 0, len(byKey))
	for _, entry := range byKey {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return R2PublishResult{
		ResourcesBaseURL: resourcesBaseURL,
		Revision:         options.Revision,
		ResourceVersion:  options.ResourceVersion,
		Entries:          entries,
	}, nil
}

func availablePublishPlatforms(baseDir, revision string) []string {
	var platforms []string
	for _, platform := range []string{"android", "ios"} {
		if _, err := os.Stat(filepath.Join(baseDir, "assets", "revisions", revision, platform, "list.bin")); err == nil {
			platforms = append(platforms, platform)
		}
	}
	if _, err := os.Stat(filepath.Join(baseDir, "assets", "revisions", revision, "list.bin")); err == nil {
		platforms = append(platforms, "")
	}
	return platforms
}

func selectPublishCandidate(candidates []assetCandidate, listSize int64) (source string, size int64, md5sum string, ok bool) {
	for _, candidate := range candidates {
		f, err := os.Open(candidate.Path)
		if err != nil {
			continue
		}
		info, err := f.Stat()
		f.Close()
		if err != nil || info.IsDir() {
			continue
		}
		if listSize >= 256 && info.Size() != listSize {
			continue
		}
		actualMD5, err := fileMD5Hex(candidate.Path, info)
		if err != nil {
			continue
		}
		if candidate.ExpectedMD5 != "" && !strings.EqualFold(candidate.ExpectedMD5, actualMD5) {
			continue
		}
		return candidate.Path, info.Size(), actualMD5, true
	}
	return "", 0, "", false
}

func safeObjectID(objectID string) bool {
	if objectID == "" || objectID == "." || objectID == ".." {
		return false
	}
	for _, r := range objectID {
		if r <= 0x1f || r == 0x7f || r == '/' || r == '\\' {
			return false
		}
	}
	return true
}

func publishPlatformName(platform string) string {
	if platform == "" {
		return "shared"
	}
	return platform
}

func requireEmptyDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	if err != nil {
		return fmt.Errorf("inspect output directory: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory %s must be empty", dir)
	}
	return nil
}

func linkOrCopy(source, target string) error {
	if err := os.Link(source, target); err == nil {
		return nil
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
