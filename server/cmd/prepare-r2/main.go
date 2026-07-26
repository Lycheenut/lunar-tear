package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"lunar-tear/server/internal/service"
)

func main() {
	baseDir := flag.String("assets-dir", ".", "root directory containing the assets/ tree")
	outputDir := flag.String("output", "r2-publish", "empty output directory to create")
	revision := flag.String("revision", "0", "asset revision to publish")
	resourceVersion := flag.String("resource-version", "", "resource URL version used in unso-<version>-<type> paths")
	resourcesBaseURL := flag.String("resources-base-url", "", "R2 custom-domain base URL embedded in list.bin")
	dryRun := flag.Bool("dry-run", false, "validate and print the plan without creating files")
	workers := flag.Int("workers", service.DefaultR2PublishWorkers, "number of files to validate concurrently")
	flag.Parse()
	if *workers < 1 {
		log.Fatal("--workers must be at least 1")
	}

	fmt.Fprintf(os.Stderr, "preparing R2 publish plan with %d workers\n", *workers)
	progress := newProgressReporter(os.Stderr)
	result, err := service.PrepareR2Publish(service.R2PublishOptions{
		BaseDir:          *baseDir,
		OutputDir:        *outputDir,
		Revision:         *revision,
		ResourceVersion:  *resourceVersion,
		ResourcesBaseURL: *resourcesBaseURL,
		DryRun:           *dryRun,
		Workers:          *workers,
		OnProgress:       progress.Report,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("validated %d R2 objects\n", len(result.Entries))
	fmt.Printf("resources base URL: %s\n", result.ResourcesBaseURL)
	if !*dryRun {
		fmt.Printf("publish directory: %s\n", *outputDir)
	}
}

type progressReporter struct {
	out         io.Writer
	phase       string
	started     time.Time
	lastPrinted time.Time
}

func newProgressReporter(out io.Writer) *progressReporter {
	return &progressReporter{out: out}
}

func (r *progressReporter) Report(progress service.R2PublishProgress) {
	now := time.Now()
	if progress.Phase != r.phase {
		r.phase = progress.Phase
		r.started = now
		r.lastPrinted = time.Time{}
	}
	if progress.Completed != 0 &&
		progress.Completed != progress.Total &&
		!r.lastPrinted.IsZero() &&
		now.Sub(r.lastPrinted) < 2*time.Second {
		return
	}

	elapsed := now.Sub(r.started)
	percent := 0.0
	if progress.Total > 0 {
		percent = float64(progress.Completed) * 100 / float64(progress.Total)
	}
	eta := "calculating"
	if progress.Completed > 0 && progress.Completed < progress.Total {
		remaining := time.Duration(
			float64(elapsed) * float64(progress.Total-progress.Completed) / float64(progress.Completed),
		)
		eta = formatProgressDuration(remaining)
	} else if progress.Completed == progress.Total {
		eta = "0s"
	}

	fmt.Fprintf(
		r.out,
		"%s: %d/%d (%.1f%%), %.2f GiB, elapsed %s, ETA %s\n",
		progress.Phase,
		progress.Completed,
		progress.Total,
		percent,
		float64(progress.BytesProcessed)/(1<<30),
		formatProgressDuration(elapsed),
		eta,
	)
	r.lastPrinted = now
}

func formatProgressDuration(duration time.Duration) string {
	if duration < time.Second {
		return "<1s"
	}
	return duration.Round(time.Second).String()
}
