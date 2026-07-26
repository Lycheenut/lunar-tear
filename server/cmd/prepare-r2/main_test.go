package main

import (
	"bytes"
	"strings"
	"testing"

	"lunar-tear/server/internal/service"
)

func TestProgressReporterPrintsStartAndCompletion(t *testing.T) {
	var output bytes.Buffer
	reporter := newProgressReporter(&output)

	reporter.Report(service.R2PublishProgress{
		Phase: service.R2PublishPhaseValidate,
		Total: 2,
	})
	reporter.Report(service.R2PublishProgress{
		Phase:          service.R2PublishPhaseValidate,
		Completed:      2,
		Total:          2,
		BytesProcessed: 1 << 30,
	})

	got := output.String()
	for _, want := range []string{
		"validate: 0/2 (0.0%)",
		"validate: 2/2 (100.0%), 1.00 GiB",
		"ETA 0s",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output %q does not contain %q", got, want)
		}
	}
}
