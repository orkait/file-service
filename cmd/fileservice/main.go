package main

import (
	"file-service/internal/app"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: Error loading .env file")
	}

	log.SetOutput(os.Stderr)

	// Initialize and run the service
	service, err := app.NewService()
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}
}
