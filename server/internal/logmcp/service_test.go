package logmcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeSource struct {
	filter  string
	limit   int
	entries []RawEntry
	err     error
}

func (f *fakeSource) Query(_ context.Context, filter string, limit int) ([]RawEntry, error) {
	f.filter = filter
	f.limit = limit
	return f.entries, f.err
}

func (*fakeSource) Close() error { return nil }

func TestSearchLogsBuildsGcplogsFilterAndRedactsSecrets(t *testing.T) {
	source := &fakeSource{entries: []RawEntry{
		{
			Timestamp: time.Date(2026, 8, 16, 1, 59, 0, 0, time.UTC),
			Severity:  "DEFAULT",
			Payload: map[string]any{
				"message": "request failed token=top-secret Authorization: Bearer abc.def",
				"container": map[string]any{
					"name": "lunar-tear-server-1",
					"id":   "1234567890abcdef",
					"metadata": map[string]any{
						"environment": "production",
						"service":     "lunar-tear-server",
					},
				},
			},
		},
	}}
	service, err := NewService("example-project", source)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 8, 16, 2, 0, 0, 0, time.UTC) }

	output, err := service.SearchLogs(context.Background(), SearchLogsInput{
		Service: "lunar-tear-server",
		Query:   "failed a.b",
		Limit:   2,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`logName="projects/example-project/logs/gcplogs-docker-driver"`,
		`jsonPayload.container.metadata.environment="production"`,
		`jsonPayload.container.metadata.service="lunar-tear-server"`,
		`jsonPayload.message=~"(?i)failed a\\.b"`,
	} {
		if !strings.Contains(source.filter, want) {
			t.Errorf("filter %q does not contain %q", source.filter, want)
		}
	}
	if strings.Contains(source.filter, "labels.environment") {
		t.Errorf("filter uses the wrong labels.environment field: %q", source.filter)
	}
	if source.limit != 3 {
		t.Fatalf("source limit = %d, want 3", source.limit)
	}
	if got := output.Entries[0].ContainerID; got != "1234567890ab" {
		t.Errorf("container ID = %q", got)
	}
	if got := output.Entries[0].Message; got != "request failed token=[REDACTED] Authorization: Bearer [REDACTED]" {
		t.Errorf("redacted message = %q", got)
	}
}

func TestSearchLogsValidatesServiceRangeAndLimit(t *testing.T) {
	service, err := NewService("example-project", &fakeSource{})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		input SearchLogsInput
		want  string
	}{
		{
			name:  "service allowlist",
			input: SearchLogsInput{Service: "other"},
			want:  "service must be one of",
		},
		{
			name: "maximum range",
			input: SearchLogsInput{
				Service:   "lunar-tear-auth",
				StartTime: "2026-08-16T00:00:00Z",
				EndTime:   "2026-08-16T02:00:00Z",
			},
			want: "time range cannot exceed",
		},
		{
			name:  "maximum limit",
			input: SearchLogsInput{Service: "lunar-tear-auth", Limit: 201},
			want:  "limit must be between",
		},
		{
			name:  "maximum query",
			input: SearchLogsInput{Service: "lunar-tear-auth", Query: strings.Repeat("a", 257)},
			want:  "query cannot exceed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.SearchLogs(context.Background(), test.input)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestGetLogContextReturnsChronologicalEntries(t *testing.T) {
	source := &fakeSource{entries: []RawEntry{
		{Timestamp: time.Date(2026, 8, 16, 2, 0, 1, 0, time.UTC), Payload: "newer"},
		{Timestamp: time.Date(2026, 8, 16, 1, 59, 59, 0, time.UTC), Payload: "older"},
	}}
	service, err := NewService("example-project", source)
	if err != nil {
		t.Fatal(err)
	}

	output, err := service.GetLogContext(context.Background(), GetLogContextInput{
		Service:   "lunar-tear-cdn",
		Timestamp: "2026-08-16T02:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := output.Entries[0].Message; got != "older" {
		t.Fatalf("first message = %q, want older", got)
	}
	if output.StartTime != "2026-08-16T01:59:30Z" || output.EndTime != "2026-08-16T02:00:30Z" {
		t.Fatalf("unexpected context range: %s to %s", output.StartTime, output.EndTime)
	}
}

func TestSearchLogsMarksExtraEntryAsTruncated(t *testing.T) {
	source := &fakeSource{entries: []RawEntry{
		{Timestamp: time.Now(), Payload: "one"},
		{Timestamp: time.Now(), Payload: "two"},
	}}
	service, err := NewService("example-project", source)
	if err != nil {
		t.Fatal(err)
	}

	output, err := service.SearchLogs(context.Background(), SearchLogsInput{
		Service: "lunar-tear-auth",
		Limit:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !output.Truncated || len(output.Entries) != 1 {
		t.Fatalf("truncated = %v, entries = %d", output.Truncated, len(output.Entries))
	}
}

func TestMCPServerAdvertisesReadOnlyTools(t *testing.T) {
	service, err := NewService("example-project", &fakeSource{})
	if err != nil {
		t.Fatal(err)
	}
	server := NewMCPServer(service)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		got = append(got, tool.Name)
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %q is not marked read-only", tool.Name)
		}
	}
	want := []string{"get_log_context", "list_services", "search_logs"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("tools = %v, want %v", got, want)
	}
}
