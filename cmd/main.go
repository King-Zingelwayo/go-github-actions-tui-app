package main

import (
	"fmt"
	"indlovu-pipeline/internal/config"
	"indlovu-pipeline/internal/ui"
	"log"
	"os"
)

func main() {
	// Load environment variables from .env file
	if err := config.LoadEnv(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}

	fmt.Println("🐘 Welcome to Indlovu Pipeline Generator")
	fmt.Println("   Indlovu's CI/CD Pipeline Setup Tool")
	fmt.Println()

	cfg := config.NewConfig()

	// Step 1: GitHub Configuration
	fmt.Println("Step 1: GitHub Configuration")
	if err := ui.GitHubConfigForm(cfg); err != nil {
		log.Fatal("GitHub configuration failed:", err)
	}

	// Step 2: AWS Configuration  
	fmt.Println("\nStep 2: AWS Configuration")
	if err := ui.AWSConfigForm(cfg); err != nil {
		log.Fatal("AWS configuration failed:", err)
	}

	// Step 3: Repository Settings
	fmt.Println("\nStep 3: Repository Settings")
	if err := ui.RepoConfigForm(cfg); err != nil {
		log.Fatal("Repository configuration failed:", err)
	}

	// Step 4: Confirmation
	fmt.Println("\nStep 4: Review & Confirm")
	confirm, err := ui.ConfirmationForm(cfg)
	if err != nil {
		log.Fatal("Confirmation failed:", err)
	}

	if !confirm {
		fmt.Println("❌ Pipeline creation cancelled")
		os.Exit(0)
	}

	// Step 5: Create Pipeline
	fmt.Println("\nStep 5: Creating Pipeline")
	pm := ui.NewPipelineManager(cfg)
	if err := pm.CreatePipeline(); err != nil {
		log.Fatal("Pipeline creation failed:", err)
	}

	fmt.Println("\n🎉 Indlovu Pipeline created successfully!")
	fmt.Println("   Your Ubuntu-powered CI/CD is ready!")
}