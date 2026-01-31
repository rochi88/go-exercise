package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"go-boilerplate/internal/app/config"
)

// Colors for output
const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[0;34m"
	colorNC     = "\033[0m" // No Color
)

const (
	migrationsPath = "file://migrations"
	configPath     = "configs"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "up":
		runUp()
	case "down":
		steps := parseIntArg(2, "number of migrations to roll back", "go run cmd/migrate/main.go down 1")
		runDown(steps)
	case "force":
		version := parseIntArg(2, "version number", "go run cmd/migrate/main.go force 1")
		runForce(version)
	case "version":
		showVersion()
	case "create":
		name := getStringArg(2, "migration name", "go run cmd/migrate/main.go create add_user_profile")
		createMigration(name)
	case "status":
		showStatus()
	case "reset":
		runReset()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Migration Helper Tool")
	fmt.Println("")
	fmt.Println("Usage: go run cmd/migrate/main.go <command> [arguments]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  up                    - Run all pending migrations")
	fmt.Println("  down <n>              - Roll back n migrations")
	fmt.Println("  force <version>       - Force set migration version (use with caution)")
	fmt.Println("  version               - Show current migration version")
	fmt.Println("  create <name>         - Create a new migration file")
	fmt.Println("  status                - Show migration status")
	fmt.Println("  reset                 - Reset database (down all + up all)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/migrate/main.go up                           # Apply all pending migrations")
	fmt.Println("  go run cmd/migrate/main.go down 1                       # Roll back 1 migration")
	fmt.Println("  go run cmd/migrate/main.go create add_user_profile      # Create new migration")
	fmt.Println("  go run cmd/migrate/main.go force 1                      # Force version to 1")
	fmt.Println("")
	fmt.Println("Configuration:")
	fmt.Println("  Database URL is loaded from config.yaml (db_url)")
	fmt.Println("  Migrations are located in the 'migrations' directory")
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.DBURL == "" {
		return nil, fmt.Errorf("database URL not configured in config.yaml")
	}

	return cfg, nil
}

func createMigrate() (*migrate.Migrate, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	printInfo("Database URL: " + cfg.DBURL)
	printInfo("Migrations Path: " + migrationsPath)
	fmt.Println("")

	m, err := migrate.New(migrationsPath, cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}

func runUp() {
	printSuccess("Running all pending migrations...")

	m, err := createMigrate()
	if err != nil {
		fatalError("Error: " + err.Error())
	}
	defer m.Close()

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			printInfo("No new migrations to apply")
		} else {
			fatalError("Migration failed: " + err.Error())
		}
	} else {
		printSuccess("✅ Migrations completed successfully!")
	}
}

func runDown(steps int) {
	printWarning(fmt.Sprintf("Rolling back %d migration(s)...", steps))

	m, err := createMigrate()
	if err != nil {
		fatalError("Error: " + err.Error())
	}
	defer m.Close()

	err = m.Steps(-steps)
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			printInfo("No migrations to roll back")
		} else {
			fatalError("Rollback failed: " + err.Error())
		}
	} else {
		printSuccess("✅ Rollback completed successfully!")
	}
}

func runForce(version int) {
	printWarning(fmt.Sprintf("⚠️  Forcing migration version to %d...", version))
	printError("This can be dangerous! Make sure you know what you're doing.")

	if !confirmAction("Are you sure? (y/N): ") {
		printWarning("Operation cancelled")
		return
	}

	m, err := createMigrate()
	if err != nil {
		fatalError("Error: " + err.Error())
	}
	defer m.Close()

	err = m.Force(version)
	if err != nil {
		fatalError("Force failed: " + err.Error())
	}

	printSuccess(fmt.Sprintf("✅ Version forced to %d", version))
}

func showVersion() {
	printInfo("Current migration version:")

	m, err := createMigrate()
	if err != nil {
		fatalError("Error: " + err.Error())
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("No migrations have been applied yet")
		} else {
			fatalError("Failed to get version: " + err.Error())
		}
	} else {
		fmt.Printf("Version: %d\n", version)
		if dirty {
			printWarning("⚠️  Database is in dirty state")
		}
	}
}

func createMigration(name string) {
	printSuccess("Creating new migration: " + name)

	// This functionality would require using the migrate CLI tool
	// For now, we'll print instructions
	printInfo("To create a new migration, use the migrate CLI tool:")
	fmt.Printf("migrate create -ext sql -dir migrations -seq %s\n", name)
	fmt.Println("")
	printInfo("Next steps:")
	fmt.Println("1. Edit the .up.sql file with your schema changes")
	fmt.Println("2. Edit the .down.sql file with the reverse changes")
	fmt.Println("3. Run: go run cmd/migrate/main.go up")
}

func showStatus() {
	printInfo("Migration status:")
	showVersion()

	fmt.Println("")
	printInfo("Available migration files:")

	// List migration files
	files, err := os.ReadDir("migrations")
	if err != nil {
		printWarning("No migration files found or unable to read migrations directory")
		return
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sql") {
			fmt.Printf("  %s\n", file.Name())
		}
	}
}

func runReset() {
	printError("⚠️  This will reset your database (drop all tables and recreate)")
	printError("This operation cannot be undone!")

	if !confirmAction("Are you sure you want to reset the database? (y/N): ") {
		printWarning("Operation cancelled")
		return
	}

	printWarning("Rolling back all migrations...")

	m, err := createMigrate()
	if err != nil {
		fatalError("Error: " + err.Error())
	}
	defer m.Close()

	// Drop all migrations
	err = m.Drop()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		printWarning("Warning: Failed to drop all migrations: " + err.Error())
		// Continue anyway, as the database might be empty
	}

	// Run all migrations up
	printSuccess("Running all migrations...")
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fatalError("Failed to run migrations: " + err.Error())
	}

	printSuccess("✅ Database reset completed!")
}

// Helper functions for colored output
func printError(message string) {
	fmt.Printf("%s%s%s\n", colorRed, message, colorNC)
}

func printSuccess(message string) {
	fmt.Printf("%s%s%s\n", colorGreen, message, colorNC)
}

func printWarning(message string) {
	fmt.Printf("%s%s%s\n", colorYellow, message, colorNC)
}

func printInfo(message string) {
	fmt.Printf("%s%s%s\n", colorBlue, message, colorNC)
}

// Helper function for user confirmation
func confirmAction(prompt string) bool {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fatalError("Failed to read input: " + err.Error())
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// Helper function for fatal errors
func fatalError(message string) {
	printError(message)
	os.Exit(1)
}

// Helper function to validate and parse integer argument
func parseIntArg(argIndex int, argName, example string) int {
	if len(os.Args) <= argIndex {
		fatalError(fmt.Sprintf("Please specify the %s", argName))
	}

	value, err := strconv.Atoi(os.Args[argIndex])
	if err != nil {
		fatalError(fmt.Sprintf("Invalid %s: %s\nExample: %s", argName, os.Args[argIndex], example))
	}

	return value
}

// Helper function to validate and get string argument
func getStringArg(argIndex int, argName, example string) string {
	if len(os.Args) <= argIndex {
		fatalError(fmt.Sprintf("Please specify %s\nExample: %s", argName, example))
	}

	return os.Args[argIndex]
}
