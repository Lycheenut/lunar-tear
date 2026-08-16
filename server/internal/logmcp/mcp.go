package logmcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewMCPServer(service *Service) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "lunar-tear-production-logs",
			Title:   "Lunar Tear Production Logs",
			Version: "1.0.0",
		},
		&mcp.ServerOptions{
			Instructions: "Read-only access to Lunar Tear production container logs in Google Cloud Logging. Call list_services before searching when the service is unknown. search_logs defaults to the last 15 minutes and permits at most one hour. get_log_context retrieves nearby entries. Sensitive token-like values are redacted.",
		},
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_services",
		Description: "List the production services that may be queried.",
		Annotations: readOnlyAnnotations(),
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ ListServicesInput) (*mcp.CallToolResult, ListServicesOutput, error) {
		return nil, service.ListServices(), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_logs",
		Description: "Search recent production container logs for one allowed service. Results are newest first.",
		Annotations: readOnlyAnnotations(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input SearchLogsInput) (*mcp.CallToolResult, SearchLogsOutput, error) {
		output, err := service.SearchLogs(ctx, input)
		return nil, output, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_log_context",
		Description: "Retrieve production log entries immediately before and after a timestamp. Results are chronological.",
		Annotations: readOnlyAnnotations(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input GetLogContextInput) (*mcp.CallToolResult, GetLogContextOutput, error) {
		output, err := service.GetLogContext(ctx, input)
		return nil, output, err
	})

	return server
}

func readOnlyAnnotations() *mcp.ToolAnnotations {
	openWorld, destructive := true, false
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    true,
		IdempotentHint:  true,
		OpenWorldHint:   &openWorld,
		DestructiveHint: &destructive,
	}
}
