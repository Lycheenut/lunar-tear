package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/term"

	"lunar-tear/server/internal/auth"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

const usage = `Usage:
  account-admin create   --name NAME [--platform android|ios] [--db PATH] [--auth-db PATH]
  account-admin delete   --name NAME [--yes] [--db PATH] [--auth-db PATH]
  account-admin password --name NAME [--auth-db PATH]

Passwords are read interactively and confirmed. Use --password-stdin to read
one password line without confirmation for non-interactive automation.`

type accountInfo struct {
	Username   string
	AuthID     int64
	UserID     int64
	PlayerID   int64
	PlayerName string
}

type accountService struct {
	auth *auth.AuthStore
	game *sqlite.SQLiteStore
}

func main() {
	log.SetFlags(0)
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		log.Printf("account-admin: %v", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return errors.New("a subcommand is required")
	}

	switch args[0] {
	case "create":
		return runCreate(args[1:], stdin, stdout, stderr)
	case "delete":
		return runDelete(args[1:], stdin, stdout, stderr)
	case "password":
		return runPassword(args[1:], stdin, stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprintln(stdout, usage)
		return nil
	default:
		fmt.Fprintln(stderr, usage)
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func runCreate(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("create", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", "", "login name")
	platformName := flags.String("platform", "android", "account platform (android or ios)")
	gameDB := flags.String("db", "db/game.db", "game database path")
	authDB := flags.String("auth-db", "db/auth.db", "authentication database path")
	passwordStdin := flags.Bool("password-stdin", false, "read one password line from stdin")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("--name is required")
	}
	platform, err := parsePlatform(*platformName)
	if err != nil {
		return err
	}
	password, err := readNewPassword(stdin, stdout, *passwordStdin)
	if err != nil {
		return err
	}

	service, closeService, err := openAccountService(*gameDB, *authDB)
	if err != nil {
		return err
	}
	defer closeService()
	info, err := service.create(*name, password, platform)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Created account %q: auth_id=%d user_id=%d player_id=%d\n", info.Username, info.AuthID, info.UserID, info.PlayerID)
	return nil
}

func runDelete(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("delete", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", "", "login name")
	gameDB := flags.String("db", "db/game.db", "game database path")
	authDB := flags.String("auth-db", "db/auth.db", "authentication database path")
	yes := flags.Bool("yes", false, "delete without interactive confirmation")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("--name is required")
	}

	service, closeService, err := openAccountService(*gameDB, *authDB)
	if err != nil {
		return err
	}
	defer closeService()
	info, err := service.lookup(*name)
	if err != nil {
		return err
	}
	printAccount(stdout, info)
	if !*yes {
		fmt.Fprintf(stdout, "Type %q to permanently delete this account: ", info.Username)
		confirmation, err := readLine(stdin)
		if err != nil {
			return fmt.Errorf("read confirmation: %w", err)
		}
		if confirmation != info.Username {
			return errors.New("confirmation did not match; account was not deleted")
		}
	}
	if err := service.delete(info); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Deleted account %q.\n", info.Username)
	return nil
}

func runPassword(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("password", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", "", "login name")
	authDB := flags.String("auth-db", "db/auth.db", "authentication database path")
	passwordStdin := flags.Bool("password-stdin", false, "read one password line from stdin")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("--name is required")
	}
	password, err := readNewPassword(stdin, stdout, *passwordStdin)
	if err != nil {
		return err
	}

	authDBHandle, err := database.Open(*authDB)
	if err != nil {
		return fmt.Errorf("open auth database: %w", err)
	}
	defer authDBHandle.Close()
	authStore, err := auth.NewAuthStore(authDBHandle)
	if err != nil {
		return fmt.Errorf("initialize auth database: %w", err)
	}
	if err := authStore.UpdatePassword(*name, password); err != nil {
		return fmt.Errorf("update password for %q: %w", *name, err)
	}
	fmt.Fprintf(stdout, "Updated password for account %q.\n", *name)
	return nil
}

func openAccountService(gameDBPath, authDBPath string) (*accountService, func(), error) {
	gameDB, err := database.Open(gameDBPath)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open game database: %w", err)
	}
	if err := migrations.Up(context.Background(), gameDB); err != nil {
		gameDB.Close()
		return nil, func() {}, fmt.Errorf("migrate game database: %w", err)
	}
	authDB, err := database.Open(authDBPath)
	if err != nil {
		gameDB.Close()
		return nil, func() {}, fmt.Errorf("open auth database: %w", err)
	}
	authStore, err := auth.NewAuthStore(authDB)
	if err != nil {
		authDB.Close()
		gameDB.Close()
		return nil, func() {}, fmt.Errorf("initialize auth database: %w", err)
	}
	closeService := func() {
		database.Checkpoint(authDB)
		database.Checkpoint(gameDB)
		authDB.Close()
		gameDB.Close()
	}
	return &accountService{auth: authStore, game: sqlite.New(gameDB, nil)}, closeService, nil
}

func (s *accountService) create(name, password string, platform model.ClientPlatform) (accountInfo, error) {
	if s.auth.UserExists(name) {
		return accountInfo{}, auth.ErrUserExists
	}
	authUser, err := s.auth.CreateUser(name, password)
	if err != nil {
		return accountInfo{}, fmt.Errorf("create auth user: %w", err)
	}

	userID, err := s.game.CreateUser(uuid.NewString(), platform)
	if err != nil {
		cleanupErr := s.auth.DeleteUser(authUser.ID)
		return accountInfo{}, errors.Join(fmt.Errorf("create game user: %w", err), cleanupError(cleanupErr))
	}
	if err := s.game.SetFacebookId(userID, authUser.ID); err != nil {
		gameCleanupErr := s.game.DeleteUser(userID)
		authCleanupErr := s.auth.DeleteUser(authUser.ID)
		return accountInfo{}, errors.Join(
			fmt.Errorf("bind game user to auth user: %w", err),
			cleanupError(gameCleanupErr),
			cleanupError(authCleanupErr),
		)
	}
	user, err := s.game.LoadUser(userID)
	if err != nil {
		return accountInfo{}, fmt.Errorf("load created game user: %w", err)
	}
	return accountInfo{
		Username:   authUser.Username,
		AuthID:     authUser.ID,
		UserID:     user.UserId,
		PlayerID:   user.PlayerId,
		PlayerName: user.Profile.Name,
	}, nil
}

func (s *accountService) lookup(name string) (accountInfo, error) {
	authUser, err := s.auth.GetUser(name)
	if err != nil {
		return accountInfo{}, fmt.Errorf("find auth user %q: %w", name, err)
	}
	info := accountInfo{Username: authUser.Username, AuthID: authUser.ID}
	userID, err := s.game.GetUserByFacebookId(authUser.ID)
	if errors.Is(err, store.ErrNotFound) {
		return info, nil
	}
	if err != nil {
		return accountInfo{}, fmt.Errorf("find linked game user: %w", err)
	}
	user, err := s.game.LoadUser(userID)
	if err != nil {
		return accountInfo{}, fmt.Errorf("load linked game user: %w", err)
	}
	info.UserID = user.UserId
	info.PlayerID = user.PlayerId
	info.PlayerName = user.Profile.Name
	return info, nil
}

func (s *accountService) delete(info accountInfo) error {
	if info.UserID != 0 {
		if err := s.game.DeleteUser(info.UserID); err != nil && !errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("delete game user %d: %w", info.UserID, err)
		}
	}
	if err := s.auth.DeleteUser(info.AuthID); err != nil {
		return fmt.Errorf("game data was deleted but auth user %d could not be deleted; retry the command: %w", info.AuthID, err)
	}
	return nil
}

func parsePlatform(value string) (model.ClientPlatform, error) {
	switch value {
	case "android":
		return model.DefaultPlatform, nil
	case "ios":
		return model.ClientPlatform{OsType: model.OsTypeIOS, PlatformType: model.PlatformTypeAppStore}, nil
	default:
		return model.ClientPlatform{}, fmt.Errorf("--platform must be android or ios, got %q", value)
	}
}

func readNewPassword(stdin io.Reader, stdout io.Writer, fromStdin bool) (string, error) {
	if fromStdin {
		password, err := readLine(stdin)
		if err != nil {
			return "", fmt.Errorf("read password from stdin: %w", err)
		}
		if password == "" {
			return "", errors.New("password must not be empty")
		}
		return password, nil
	}

	file, ok := stdin.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", errors.New("stdin is not a terminal; use --password-stdin for non-interactive input")
	}
	password, err := readHiddenPassword(file, stdout, "Password: ")
	if err != nil {
		return "", err
	}
	confirmation, err := readHiddenPassword(file, stdout, "Confirm password: ")
	if err != nil {
		return "", err
	}
	if password == "" {
		return "", errors.New("password must not be empty")
	}
	if password != confirmation {
		return "", errors.New("passwords did not match")
	}
	return password, nil
}

func readHiddenPassword(file *os.File, stdout io.Writer, prompt string) (string, error) {
	fmt.Fprint(stdout, prompt)
	password, err := term.ReadPassword(int(file.Fd()))
	fmt.Fprintln(stdout)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(password), nil
}

func readLine(reader io.Reader) (string, error) {
	line, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if errors.Is(err, io.EOF) && line == "" {
		return "", io.EOF
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func printAccount(writer io.Writer, info accountInfo) {
	fmt.Fprintf(writer, "Login name: %s\n", info.Username)
	fmt.Fprintf(writer, "Auth ID:    %d\n", info.AuthID)
	if info.UserID == 0 {
		fmt.Fprintln(writer, "Game data:  no linked game user")
		return
	}
	fmt.Fprintf(writer, "User ID:    %d\n", info.UserID)
	fmt.Fprintf(writer, "Player ID:  %d\n", info.PlayerID)
	fmt.Fprintf(writer, "Player name: %s\n", info.PlayerName)
}

func cleanupError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("cleanup failed: %w", err)
}
