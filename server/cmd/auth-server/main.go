package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"lunar-tear/server/internal/auth"
	"lunar-tear/server/internal/database"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:3000", "HTTP listen address (host:port)")
	dbPath := flag.String("db", "db/auth.db", "SQLite database path for auth users")
	secret := flag.String("secret", "", "HMAC secret for tokens (minimum 32 bytes; prefer --secret-file)")
	secretFile := flag.String("secret-file", "", "path to a persistent hex-encoded HMAC secret (default: auth.secret next to --db)")
	allowedRedirects := flag.String("allowed-redirect-uris", os.Getenv("LUNAR_AUTH_REDIRECT_URIS"), "comma-separated exact OAuth callback URIs; production deployments should set this")
	noRegister := flag.Bool("no-register", false, "Disallow new account registrations for clients, when present. Default = false")
	flag.Parse()

	if *secretFile == "" {
		*secretFile = filepath.Join(filepath.Dir(*dbPath), "auth.secret")
	}
	hmacSecret, generated, err := loadTokenSecret(*secret, *secretFile)
	if err != nil {
		log.Fatalf("load token secret: %v", err)
	}
	if generated {
		log.Printf("generated persistent token secret at %s", *secretFile)
	}

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	db.SetMaxOpenConns(1)
	defer func() {
		database.Checkpoint(db)
		db.Close()
	}()

	store, err := auth.NewAuthStore(db)
	if err != nil {
		log.Fatalf("init auth store: %v", err)
	}

	tok := auth.NewTokenService(hmacSecret)
	h := NewHandlers(store, tok, *noRegister, splitRedirectURIs(*allowedRedirects))

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.HandleOAuth)
	mux.HandleFunc("/me", h.HandleMe)
	mux.HandleFunc("/check-username", h.HandleCheckUsername)

	srv := &http.Server{
		Addr:              *listen,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("auth server listening on %s", *listen)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("listen: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func splitRedirectURIs(raw string) []string {
	var values []string
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func loadTokenSecret(explicit, secretFile string) (secret []byte, generated bool, err error) {
	if explicit != "" {
		if len(explicit) < 32 {
			return nil, false, fmt.Errorf("--secret must contain at least 32 bytes")
		}
		return []byte(explicit), false, nil
	}
	if secretFile == "" {
		return nil, false, fmt.Errorf("--secret-file is required when --secret is empty")
	}

	read := func() ([]byte, error) {
		data, readErr := os.ReadFile(secretFile)
		if readErr != nil {
			return nil, readErr
		}
		decoded, decodeErr := hex.DecodeString(strings.TrimSpace(string(data)))
		if decodeErr != nil {
			return nil, fmt.Errorf("decode %s: %w", secretFile, decodeErr)
		}
		if len(decoded) < 32 {
			return nil, fmt.Errorf("%s must contain at least 32 decoded bytes", secretFile)
		}
		return decoded, nil
	}

	if existing, readErr := read(); readErr == nil {
		return existing, false, nil
	} else if !os.IsNotExist(readErr) {
		return nil, false, readErr
	}

	if err := os.MkdirAll(filepath.Dir(secretFile), 0700); err != nil {
		return nil, false, fmt.Errorf("create secret directory: %w", err)
	}
	generatedSecret := make([]byte, 32)
	if _, err := rand.Read(generatedSecret); err != nil {
		return nil, false, fmt.Errorf("generate secret: %w", err)
	}
	f, err := os.OpenFile(secretFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			existing, readErr := read()
			return existing, false, readErr
		}
		return nil, false, fmt.Errorf("create %s: %w", secretFile, err)
	}
	if _, err := fmt.Fprintln(f, hex.EncodeToString(generatedSecret)); err != nil {
		f.Close()
		return nil, false, fmt.Errorf("write %s: %w", secretFile, err)
	}
	if err := f.Close(); err != nil {
		return nil, false, fmt.Errorf("close %s: %w", secretFile, err)
	}
	if err := os.Chmod(secretFile, 0600); err != nil {
		return nil, false, fmt.Errorf("secure %s: %w", secretFile, err)
	}
	return generatedSecret, true, nil
}
