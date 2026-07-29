package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"lunar-tear/server/internal/service"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:8080", "local bind address (host:port)")
	publicAddr := flag.String("public-addr", "127.0.0.1:8080", "externally-reachable host:port used for list.bin URL rewriting")
	resourcesBaseURLFlag := flag.String("resources-base-url", "", "public HTTP(S) resource base URL embedded in list.bin; padded to the required 43 bytes")
	assetsDir := flag.String("assets-dir", ".", "root directory containing the assets/ tree")
	flag.Parse()

	resourcesBaseURLInput := *resourcesBaseURLFlag
	if resourcesBaseURLInput == "" {
		resourcesBaseURLInput = "http://" + *publicAddr
	}
	resourcesBaseURL, err := service.NormalizeResourcesBaseURL(resourcesBaseURLInput)
	if err != nil {
		log.Fatalf("[config] resources base URL: %v", err)
	}
	log.Printf("[config] resources base URL: %s", resourcesBaseURL)

	octoServer := service.NewOctoHTTPServer(resourcesBaseURL, *assetsDir)
	h2s := &http2.Server{}
	handler := h2c.NewHandler(octoServer.Handler(), h2s)

	srv := &http.Server{
		Addr:    *listen,
		Handler: handler,
	}
	http2.ConfigureServer(srv, h2s)

	// Resolve actual listen address for logging (useful when port is 0).
	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *listen, err)
	}
	log.Printf("Octo CDN listening on %s (HTTP/1.1 + h2c)", lis.Addr())
	log.Printf("public address: %s", *publicAddr)
	if *assetsDir != "." {
		log.Printf("assets directory: %s", *assetsDir)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Serve(lis); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	srv.Shutdown(context.Background())
	log.Println("shutdown complete")
}
