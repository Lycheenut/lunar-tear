package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"lunar-tear/server/internal/logmcp"
)

func main() {
	project := flag.String("project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "Google Cloud project containing production logs")
	flag.Parse()
	projectID := strings.TrimSpace(*project)
	if projectID == "" {
		log.Fatal("Google Cloud project is required; pass --project or set GOOGLE_CLOUD_PROJECT")
	}

	ctx := context.Background()
	source, err := logmcp.NewCloudSource(ctx, projectID)
	if err != nil {
		log.Fatalf("create Cloud Logging client: %v", err)
	}
	defer func() {
		if err := source.Close(); err != nil {
			log.Printf("close Cloud Logging client: %v", err)
		}
	}()

	service, err := logmcp.NewService(projectID, source)
	if err != nil {
		log.Fatal(err)
	}
	if err := logmcp.NewMCPServer(service).Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("run MCP server: %v", err)
	}
}
