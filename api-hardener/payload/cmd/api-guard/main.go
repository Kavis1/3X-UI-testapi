//go:build toolsignore
// +build toolsignore

package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/mhsanaei/3x-ui/v2/config"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/web/service"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "install":
		return handleInstall(args[1:])
	case "create":
		return handleCreate(args[1:])
	case "list":
		return handleList()
	case "enable":
		return handleToggle(args[1:], true)
	case "disable":
		return handleToggle(args[1:], false)
	case "delete":
		return handleDelete(args[1:])
	case "rotate":
		return handleRotate(args[1:])
	case "rate":
		return handleRate(args[1:])
	default:
		printUsage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage() {
	fmt.Println("Usage: api-guard <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  install      Apply API hardening (migrate DB, enforce tokens, create bootstrap user)")
	fmt.Println("  create       Create a new API user and print its token")
	fmt.Println("  list         List API users with status and rate limits")
	fmt.Println("  enable       Enable an API user by id")
	fmt.Println("  disable      Disable an API user by id")
	fmt.Println("  delete       Delete an API user by id")
	fmt.Println("  rotate       Rotate token for an API user and print the new token")
	fmt.Println("  rate         Set per-minute rate limit for an API user (0 = unlimited)")
}

func initDB() error {
	return database.InitDB(config.GetDBPath())
}

func handleInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	tokenOnly := fs.Bool("token-only", true, "deny session-based access to /panel/api and require tokens")
	defaultRate := fs.Int("default-rate", 120, "default per-minute limit for API tokens (0 = unlimited)")
	bootstrapUser := fs.String("bootstrap-user", "api-root", "bootstrap API user (created only when none exist)")
	bootstrapRate := fs.Int("bootstrap-rate", 120, "rate limit for the bootstrap user (0 = use default)")
	fs.Parse(args)

	if err := initDB(); err != nil {
		return err
	}
	defer database.CloseDB()

	settingSvc := service.SettingService{}
	apiSvc := service.APIUserService{}

	if err := settingSvc.SetAPIDefaultRateLimit(*defaultRate); err != nil {
		return err
	}
	if *tokenOnly {
		if err := settingSvc.SetAPITokenOnly(true); err != nil {
			return err
		}
	}

	count, err := apiSvc.Count()
	if err != nil {
		return err
	}

	if count == 0 {
		user, token, err := apiSvc.CreateUser(*bootstrapUser, *bootstrapRate)
		if err != nil {
			return err
		}
		fmt.Printf("Bootstrap API user created (id=%d, name=%s)\n", user.Id, user.Name)
		fmt.Printf("Token (store securely, shown once): %s\n", token)
	} else {
		fmt.Printf("API users already present (%d); bootstrap user not created\n", count)
	}

	fmt.Println("API hardening installed.")
	return nil
}

func handleCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	name := fs.String("name", "", "API user name (required)")
	rate := fs.Int("rate", 0, "per-minute rate limit (0 = use default)")
	fs.Parse(args)

	if *name == "" {
		return fmt.Errorf("-name is required")
	}

	if err := initDB(); err != nil {
		return err
	}
	defer database.CloseDB()

	apiSvc := service.APIUserService{}
	user, token, err := apiSvc.CreateUser(*name, *rate)
	if err != nil {
		return err
	}

	fmt.Printf("API user created (id=%d, name=%s, rate=%d/min)\n", user.Id, user.Name, user.RateLimitPerMinute)
	fmt.Printf("Token (store securely, shown once): %s\n", token)
	return nil
}

func handleList() error {
	if err := initDB(); err != nil {
		return err
	}
	defer database.CloseDB()

	apiSvc := service.APIUserService{}
	users, err := apiSvc.ListUsers()
	if err != nil {
		return err
	}

	if len(users) == 0 {
		fmt.Println("No API users found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tENABLED\tRATE/MIN\tLAST USED")
	for _, u := range users {
		lastUsed := "never"
		if u.LastUsedAt != nil {
			lastUsed = u.LastUsedAt.Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%d\t%s\t%t\t%d\t%s\n", u.Id, u.Name, u.Enabled, u.RateLimitPerMinute, lastUsed)
	}
	w.Flush()
	return nil
}

func handleToggle(args []string, enabled bool) error {
	fs := flag.NewFlagSet("toggle", flag.ExitOnError)
	id := fs.Int("id", 0, "API user id")
	fs.Parse(args)

	if *id <= 0 {
		return fmt.Errorf("-id must be provided")
	}

	if err := initDB(); err != nil {
		return err
	}
	defer database.CloseDB()

	apiSvc := service.APIUserService{}
	if err := apiSvc.SetEnabled(*id, enabled); err != nil {
		return err
	}
	if enabled {
		fmt.Printf("API user %d enabled\n", *id)
	} else {
		fmt.Printf("API user %d disabled\n", *id)
	}
	return nil
}

func handleDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	id := fs.Int("id", 0, "API user id")
	fs.Parse(args)

	if *id <= 0 {
		return fmt.Errorf("-id must be provided")
	}

	if err := initDB(); err != nil {
		return err
	}
	defer database.CloseDB()

	apiSvc := service.APIUserService{}
	if err := apiSvc.DeleteUser(*id); err != nil {
		return err
	}
	fmt.Printf("API user %d deleted\n", *id)
	return nil
}

func handleRotate(args []string) error {
	fs := flag.NewFlagSet("rotate", flag.ExitOnError)
	id := fs.Int("id", 0, "API user id")
	fs.Parse(args)

	if *id <= 0 {
		return fmt.Errorf("-id must be provided")
	}

	if err := initDB(); err != nil {
		return err
	}
	defer database.CloseDB()

	apiSvc := service.APIUserService{}
	token, err := apiSvc.RotateToken(*id)
	if err != nil {
		return err
	}
	fmt.Printf("API user %d token rotated\n", *id)
	fmt.Printf("New token (store securely, shown once): %s\n", token)
	return nil
}

func handleRate(args []string) error {
	fs := flag.NewFlagSet("rate", flag.ExitOnError)
	id := fs.Int("id", 0, "API user id")
	rate := fs.Int("rate", 0, "per-minute rate limit (0 = unlimited/default)")
	fs.Parse(args)

	if *id <= 0 {
		return fmt.Errorf("-id must be provided")
	}

	if err := initDB(); err != nil {
		return err
	}
	defer database.CloseDB()

	apiSvc := service.APIUserService{}
	if err := apiSvc.UpdateRateLimit(*id, *rate); err != nil {
		return err
	}
	fmt.Printf("API user %d rate limit set to %d requests/minute\n", *id, *rate)
	return nil
}
