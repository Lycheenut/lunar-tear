package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

const DefaultR2PublishWorkers = 4

const (
	R2PublishPhaseValidate    = "validate"
	R2PublishPhaseMaterialize = "materialize"
)

type R2PublishProgress struct {
	Phase          string
	Completed      int
	Total          int
	BytesProcessed int64
}

type R2PublishOptions struct {
	BaseDir          string
	OutputDir        string
	Revision         string
	ResourceVersion  string
	ResourcesBaseURL string
	DryRun           bool
	Workers          int
	OnProgress       func(R2PublishProgress)
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
	if !options.DryRun {
		if options.OutputDir == "" {
			return R2PublishResult{}, fmt.Errorf("output directory is required")
		}
		if err := requireEmptyDirectory(options.OutputDir); err != nil {
			return R2PublishResult{}, err
		}
	}

	result, err := buildR2PublishPlan(options)
	if err != nil || options.DryRun {
		return result, err
	}

	var materializedBytes int64
	reportR2PublishProgress(options, R2PublishPhaseMaterialize, 0, len(result.Entries), 0)
	for i, entry := range result.Entries {
		target := filepath.Join(options.OutputDir, filepath.FromSlash(entry.Key))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return result, fmt.Errorf("create output directory for %s: %w", entry.Key, err)
		}
		source := filepath.Join(options.BaseDir, filepath.FromSlash(entry.Source))
		if err := linkOrCopy(source, target); err != nil {
			return result, fmt.Errorf("materialize %s: %w", entry.Key, err)
		}
		materializedBytes += entry.Size
		reportR2PublishProgress(options, R2PublishPhaseMaterialize, i+1, len(result.Entries), materializedBytes)
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

	var jobs []r2PublishJob
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
			jobs = append(jobs, r2PublishJob{
				platform: platform,
				objectID: objectID,
			})
		}
	}
	if !options.DryRun {
		first, second, collision := firstCaseInsensitiveObjectIDCollision(jobs)
		if collision {
			if err := ensureCaseSensitiveOutputDirectory(options.OutputDir); err != nil {
				return R2PublishResult{}, fmt.Errorf(
					"R2 object IDs %q and %q differ only by case: %w",
					first,
					second,
					err,
				)
			}
		}
	}

	jobResults := runR2PublishJobs(options, keyPrefix, jobs)
	byKey := make(map[string]R2PublishEntry)
	var missing []string
	var collisions []string
	for _, jobResult := range jobResults {
		if jobResult.err != nil {
			return R2PublishResult{}, jobResult.err
		}
		if jobResult.missing != "" {
			missing = append(missing, jobResult.missing)
		}
		for _, entry := range jobResult.entries {
			if previous, exists := byKey[entry.Key]; exists {
				if previous.Size != entry.Size || !strings.EqualFold(previous.MD5, entry.MD5) {
					collisions = append(collisions, fmt.Sprintf(
						"%s differs between %s (%s) and %s (%s)",
						entry.Key, previous.Platform, previous.MD5, entry.Platform, entry.MD5,
					))
				}
				continue
			}
			byKey[entry.Key] = entry
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

type r2PublishJob struct {
	platform string
	objectID string
}

type r2PublishJobResult struct {
	entries []R2PublishEntry
	missing string
	err     error
	bytes   int64
}

type indexedR2PublishJob struct {
	index int
	job   r2PublishJob
}

type indexedR2PublishJobResult struct {
	index  int
	result r2PublishJobResult
}

func runR2PublishJobs(options R2PublishOptions, keyPrefix string, jobs []r2PublishJob) []r2PublishJobResult {
	results := make([]r2PublishJobResult, len(jobs))
	reportR2PublishProgress(options, R2PublishPhaseValidate, 0, len(jobs), 0)
	if len(jobs) == 0 {
		return results
	}

	workerCount := r2PublishWorkerCount(options.Workers, len(jobs))
	jobCh := make(chan indexedR2PublishJob)
	resultCh := make(chan indexedR2PublishJobResult, workerCount)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for indexedJob := range jobCh {
				resultCh <- indexedR2PublishJobResult{
					index:  indexedJob.index,
					result: resolveR2PublishJob(options, keyPrefix, indexedJob.job),
				}
			}
		}()
	}
	go func() {
		for i, job := range jobs {
			jobCh <- indexedR2PublishJob{index: i, job: job}
		}
		close(jobCh)
	}()

	var bytesProcessed int64
	for completed := 1; completed <= len(jobs); completed++ {
		indexedResult := <-resultCh
		results[indexedResult.index] = indexedResult.result
		bytesProcessed += indexedResult.result.bytes
		reportR2PublishProgress(options, R2PublishPhaseValidate, completed, len(jobs), bytesProcessed)
	}
	workers.Wait()
	return results
}

func resolveR2PublishJob(options R2PublishOptions, keyPrefix string, job r2PublishJob) r2PublishJobResult {
	if !safeObjectID(job.objectID) {
		return r2PublishJobResult{
			missing: fmt.Sprintf("%s:%q has an unsafe object ID", publishPlatformName(job.platform), job.objectID),
		}
	}

	var result r2PublishJobResult
	for _, assetType := range []string{"assetbundle", "resources"} {
		candidates, listSize, ok := objectIdToFilePathCandidates(
			options.BaseDir, options.Revision, job.platform, assetType, job.objectID,
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
			result.err = fmt.Errorf("asset source %s is outside assets directory %s", source, options.BaseDir)
			return result
		}
		result.entries = append(result.entries, R2PublishEntry{
			Key: path.Join(
				keyPrefix,
				"resource-bundle-server",
				"unso-"+options.ResourceVersion+"-"+assetType,
				job.objectID,
			),
			Platform: publishPlatformName(job.platform),
			Type:     assetType,
			ObjectID: job.objectID,
			Source:   filepath.ToSlash(relativeSource),
			Size:     size,
			MD5:      md5sum,
		})
		result.bytes += size
	}
	if len(result.entries) == 0 {
		result.missing = fmt.Sprintf("%s:%q has no valid local asset", publishPlatformName(job.platform), job.objectID)
	}
	return result
}

func r2PublishWorkerCount(requested, total int) int {
	if requested <= 0 {
		requested = DefaultR2PublishWorkers
	}
	if total > 0 {
		requested = min(requested, total)
	}
	return max(requested, 1)
}

func reportR2PublishProgress(options R2PublishOptions, phase string, completed, total int, bytesProcessed int64) {
	if options.OnProgress != nil {
		options.OnProgress(R2PublishProgress{
			Phase:          phase,
			Completed:      completed,
			Total:          total,
			BytesProcessed: bytesProcessed,
		})
	}
}

func firstCaseInsensitiveObjectIDCollision(jobs []r2PublishJob) (string, string, bool) {
	seen := make(map[string]string, len(jobs))
	for _, job := range jobs {
		folded := strings.ToLower(job.objectID)
		previous, exists := seen[folded]
		if exists && previous != job.objectID {
			return previous, job.objectID, true
		}
		if !exists {
			seen[folded] = job.objectID
		}
	}
	return "", "", false
}

func ensureCaseSensitiveOutputDirectory(dir string) error {
	supported, err := directorySupportsCaseSensitiveNames(dir)
	if err != nil {
		return fmt.Errorf("check whether output directory is case-sensitive: %w", err)
	}
	if supported {
		return nil
	}
	if runtime.GOOS != "windows" {
		return fmt.Errorf("output directory %s is on a case-insensitive filesystem", dir)
	}

	absoluteDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	output, commandErr := exec.Command(
		"fsutil.exe",
		"file",
		"setCaseSensitiveInfo",
		absoluteDir,
		"enable",
	).CombinedOutput()
	if commandErr != nil {
		return fmt.Errorf(
			"output directory must be case-sensitive; run PowerShell as Administrator and execute `fsutil.exe file setCaseSensitiveInfo \"%s\" enable` (automatic enable failed: %v: %s)",
			absoluteDir,
			commandErr,
			strings.TrimSpace(string(output)),
		)
	}

	supported, err = directorySupportsCaseSensitiveNames(dir)
	if err != nil {
		return fmt.Errorf("verify case-sensitive output directory: %w", err)
	}
	if !supported {
		return fmt.Errorf("output directory %s remains case-insensitive after enabling it", absoluteDir)
	}
	return nil
}

func directorySupportsCaseSensitiveNames(dir string) (bool, error) {
	firstPath := filepath.Join(dir, ".r2-case-probe")
	secondPath := filepath.Join(dir, ".R2-CASE-PROBE")

	first, err := os.OpenFile(firstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false, err
	}
	if err := first.Close(); err != nil {
		os.Remove(firstPath)
		return false, err
	}
	defer os.Remove(firstPath)

	second, err := os.OpenFile(secondPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := second.Close(); err != nil {
		os.Remove(secondPath)
		return false, err
	}
	defer os.Remove(secondPath)
	return true, nil
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
