package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store/sqlite"
)

const masterDataPath = "assets/release/20240404193219.bin.e"

func main() {
	listen := flag.String("listen", "0.0.0.0:443", "gRPC listen address (host:port)")
	publicAddr := flag.String("public-addr", "127.0.0.1:443", "externally-reachable host:port advertised to clients")
	dbPath := flag.String("db", "db/game.db", "SQLite database path")
	octoURL := flag.String("octo-url", "", "Octo CDN base URL the client will use for assets (e.g. http://10.0.2.2:8080)")
	authURL := flag.String("auth-url", "", "Auth server base URL for Facebook token validation (e.g. http://localhost:3000)")
	adminListen := flag.String("admin-listen", "127.0.0.1:8082", "admin UI/API listen address (host:port). Loopback by default; only binds when LUNAR_ADMIN_TOKEN is set.")
	gachaConfigPath := flag.String("gacha-config", "config/gacha.json", "server-owned plaintext Gacha configuration path")
	noRegister := flag.Bool("no-register", false, "Disallow new account registrations for clients, when present. Default = false")
	flag.Parse()

	if *octoURL == "" {
		log.Fatalf("--octo-url is required (e.g. http://10.0.2.2:8080)")
	}

	holder, err := runtime.NewHolderWithGachaConfig(masterDataPath, *gachaConfigPath)
	if err != nil {
		log.Fatalf("init master data: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	log.Printf("database opened: %s", *dbPath)

	userStore := sqlite.New(db, gametime.Now)

	grpcServer := startGRPC(*listen, *publicAddr, *octoURL, *authURL, userStore, holder, *noRegister)

	startAdmin(*adminListen, masterDataPath, *gachaConfigPath, holder)

	<-ctx.Done()
	log.Println("shutting down...")

	grpcServer.GracefulStop()
	database.Checkpoint(db)

	log.Println("shutdown complete")
}
