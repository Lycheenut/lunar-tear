package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

type service struct {
	label string
	color string
	cmd   *exec.Cmd
}

func binExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func buildAll() {
	if err := os.MkdirAll("bin", 0755); err != nil {
		log.Fatalf("create bin/: %v", err)
	}

	type target struct {
		name string
		pkg  string
	}
	targets := []target{
		{"auth-server", "./cmd/auth-server"},
		{"octo-cdn", "./cmd/octo-cdn"},
		{"lunar-tear", "./cmd/lunar-tear"},
	}

	ext := binExt()
	var wg sync.WaitGroup
	errs := make(chan error, len(targets))

	for _, t := range targets {
		wg.Add(1)
		go func(t target) {
			defer wg.Done()
			out := filepath.Join("bin", t.name+ext)
			cmd := exec.Command("go", "build", "-o", out, t.pkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				errs <- fmt.Errorf("build %s: %w", t.name, err)
			}
		}(t)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		log.Fatal(err)
	}
}

func main() {
	// auth-server flags
	authListen := flag.String("auth.listen", "0.0.0.0:3000", "auth-server listen address (host:port)")
	authDB := flag.String("auth.db", "db/auth.db", "auth-server SQLite database path")

	// octo-cdn flags
	cdnListen := flag.String("cdn.listen", "0.0.0.0:8080", "octo-cdn local bind address")
	cdnPublicAddr := flag.String("cdn.public-addr", "10.0.2.2:8080", "octo-cdn externally-reachable address")

	// lunar-tear (grpc) flags
	grpcListen := flag.String("grpc.listen", "0.0.0.0:8003", "lunar-tear gRPC listen address (host:port)")
	grpcPublicAddr := flag.String("grpc.public-addr", "10.0.2.2:8003", "lunar-tear externally-reachable address")
	grpcDB := flag.String("grpc.db", "db/game.db", "lunar-tear SQLite database path")
	grpcOctoURL := flag.String("grpc.octo-url", "", "Octo CDN base URL passed to lunar-tear (default: derived from cdn.public-addr)")
	grpcAuthURL := flag.String("grpc.auth-url", "", "auth server base URL passed to lunar-tear (default: derived from auth.listen)")

	// admin UI/API is opt-in; empty leaves lunar-tear's own default in place
	// (the listener still only binds if LUNAR_ADMIN_TOKEN is set in the env).
	adminListen := flag.String("admin.listen", "", "lunar-tear admin UI/API listen address (host:port). Empty = leave default; listener only binds when LUNAR_ADMIN_TOKEN is set in the env.")

	// Controlled server access
	noRegister := flag.Bool("no-register", false, "Disallow new account registrations for clients, when present. Default = false")

	// dev utility output config
	noColor := flag.Bool("no-color", false, "disable colored output")
	readyFile := flag.String("ready-file", "", "write the supervisor PID here after all services start")
	stopFile := flag.String("stop-file", "", "shut down when this file is created")

	flag.Parse()
	removeControlFile(*readyFile)
	removeControlFile(*stopFile)
	defer removeControlFile(*readyFile)
	defer removeControlFile(*stopFile)

	if *grpcOctoURL == "" {
		*grpcOctoURL = fmt.Sprintf("http://%s", *cdnPublicAddr)
	}
	if *grpcAuthURL == "" {
		*grpcAuthURL = fmt.Sprintf("http://%s", *authListen)
	}

	if *noColor || !colorSupported() {
		colorReset = ""
		colorRed = ""
		colorGreen = ""
		colorYellow = ""
		colorCyan = ""
	}

	if _, err := os.Stat("go.mod"); err == nil {
		log.Println("building services...")
		buildAll()
	} else {
		log.Println("prebuilt mode: skipping build, using bin/ from archive")
	}

	ext := binExt()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *stopFile != "" {
		go watchStopFile(ctx, *stopFile, stop)
	}

	noreg_s := ""
	if *noRegister {
		noreg_s = "--no-register"
	}

	services := []service{
		{
			label: "auth",
			color: colorGreen,
			cmd: exec.CommandContext(ctx, filepath.Join("bin", "auth-server"+ext),
				"--listen", *authListen,
				"--db", *authDB,
				noreg_s,
			),
		},
		{
			label: "cdn",
			color: colorCyan,
			cmd: exec.CommandContext(ctx, filepath.Join("bin", "octo-cdn"+ext),
				"--listen", *cdnListen,
				"--public-addr", *cdnPublicAddr,
			),
		},
		{
			label: "grpc",
			color: colorYellow,
			cmd: exec.CommandContext(ctx, filepath.Join("bin", "lunar-tear"+ext),
				grpcArgs(*grpcListen, *grpcPublicAddr, *grpcDB, *grpcOctoURL, *grpcAuthURL, *adminListen, *noRegister)...,
			),
		},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(services))

	for i := range services {
		svc := &services[i]
		stdout, err := svc.cmd.StdoutPipe()
		if err != nil {
			log.Printf("[%s] stdout pipe: %v", svc.label, err)
			stop()
			wg.Wait()
			return
		}
		stderr, err := svc.cmd.StderrPipe()
		if err != nil {
			log.Printf("[%s] stderr pipe: %v", svc.label, err)
			stop()
			wg.Wait()
			return
		}

		if err := svc.cmd.Start(); err != nil {
			log.Printf("[%s] start: %v", svc.label, err)
			stop()
			wg.Wait()
			return
		}

		prefix := fmt.Sprintf("%s[%s]%s ", svc.color, svc.label, colorReset)
		wg.Add(2)
		go prefixLines(&wg, prefix, stdout)
		go prefixLines(&wg, prefix, stderr)

		wg.Add(1)
		go func(s *service) {
			defer wg.Done()
			if err := s.cmd.Wait(); err != nil {
				errCh <- fmt.Errorf("[%s] %w", s.label, err)
			}
		}(svc)

		log.Printf("%s%s started (pid %d)%s", svc.color, svc.label, svc.cmd.Process.Pid, colorReset)
	}

	if err := writeReadyFile(*readyFile); err != nil {
		log.Printf("write ready file: %v", err)
		stop()
		wg.Wait()
		return
	}

	select {
	case <-ctx.Done():
		log.Println("shutting down all services...")
	case err := <-errCh:
		log.Printf("%s%s%s", colorRed, err, colorReset)
		stop()
	}

	wg.Wait()
}

func removeControlFile(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("remove control file %s: %v", path, err)
	}
}

func writeReadyFile(path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0644)
}

func watchStopFile(ctx context.Context, path string, stop context.CancelFunc) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := os.Stat(path); err == nil {
				log.Printf("stop requested through %s", path)
				stop()
				return
			} else if !os.IsNotExist(err) {
				log.Printf("check stop file %s: %v", path, err)
			}
		}
	}
}

func prefixLines(wg *sync.WaitGroup, prefix string, r io.Reader) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Printf("%s%s\n", prefix, scanner.Text())
	}
}

// grpcArgs assembles the argv for the lunar-tear subprocess. The admin flag
// is appended only when --admin.listen was supplied so we don't override
// lunar-tear's own default when the operator hasn't opted in.
func grpcArgs(listen, publicAddr, db, octoURL, authURL, adminListen string, noRegister bool) []string {
	args := []string{
		"--listen", listen,
		"--public-addr", publicAddr,
		"--db", db,
		"--octo-url", octoURL,
		"--auth-url", authURL,
	}

	if adminListen != "" {
		args = append(args, "--admin-listen", adminListen)
	}

	if noRegister {
		args = append(args, "--no-register")
	}
	return args
}
