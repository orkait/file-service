package main

import (
	"file-service/config"
	"file-service/pkg/database"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("=== Setting Up Database ===")
	fmt.Println()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Connected to database")
	fmt.Println()

	schema, err := os.ReadFile("database/schema.sql")
	if err != nil {
		log.Fatalf("❌ Failed to read schema file: %v", err)
	}

	fmt.Println("Executing schema...")
	_, err = db.DB.Exec(string(schema))
	if err != nil {
		log.Fatalf("❌ Failed to execute schema: %v", err)
	}

	fmt.Println("✅ Schema executed successfully")
	fmt.Println()

	fmt.Println("=== Verifying Tables ===")
	tables := []string{"clients", "projects", "assets", "api_keys", "project_members", "refresh_tokens"}

	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)`
		err := db.DB.QueryRow(query, table).Scan(&exists)
		if err != nil {
			fmt.Printf("❌ Error checking table '%s': %v\n", table, err)
			continue
		}

		if exists {
			fmt.Printf("✅ Table '%s' created\n", table)
		} else {
			fmt.Printf("❌ Table '%s' NOT created\n", table)
		}
	}

	fmt.Println()
	fmt.Println("=== Database Setup Complete ===")
	fmt.Println()
	fmt.Println("Next: Run 'go run main.go' to start the server")
}
