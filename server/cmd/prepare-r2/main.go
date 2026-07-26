package main

import (
	"flag"
	"fmt"
	"log"

	"lunar-tear/server/internal/service"
)

func main() {
	baseDir := flag.String("assets-dir", ".", "root directory containing the assets/ tree")
	outputDir := flag.String("output", "r2-publish", "empty output directory to create")
	revision := flag.String("revision", "0", "asset revision to publish")
	resourceVersion := flag.String("resource-version", "", "resource URL version used in unso-<version>-<type> paths")
	resourcesBaseURL := flag.String("resources-base-url", "", "R2 custom-domain base URL embedded in list.bin")
	dryRun := flag.Bool("dry-run", false, "validate and print the plan without creating files")
	flag.Parse()

	result, err := service.PrepareR2Publish(service.R2PublishOptions{
		BaseDir:          *baseDir,
		OutputDir:        *outputDir,
		Revision:         *revision,
		ResourceVersion:  *resourceVersion,
		ResourcesBaseURL: *resourcesBaseURL,
		DryRun:           *dryRun,
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
