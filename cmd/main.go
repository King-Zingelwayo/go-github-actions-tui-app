package main

import (
	_ "embed"
	"fmt"
	"indlovu-pipeline/internal/config"
	"indlovu-pipeline/internal/ui"
	"log"
	"os"
	"flag"
	"strings"
)

//go:embed .env
var embeddedEnv string
const version = "v1.2.0"

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (shorthand)")
	flag.Parse()

	if showVersion {
		fmt.Printf("🐘 Elephant TF CI %s\n", version)
		fmt.Println("Ubuntu-powered Terraform CI/CD Pipeline Generator")
		os.Exit(0)
	}

	// Load embedded .env first
	loadEmbeddedEnv()
	
	// Then try to load external .env (will override embedded if exists)
	if err := config.LoadEnv(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}

	fmt.Println("🐘 Welcome to Elephant TF CI")
	fmt.Println("   Ubuntu-powered Terraform CI/CD Pipeline Generator")
	fmt.Println()

	// Authenticate and select repository once at startup
	cfg := config.NewConfig()
	fmt.Println("🔐 GitHub Authentication Required")
	if err := ui.AuthenticateGitHub(cfg); err != nil {
		log.Fatal("Authentication failed:", err)
	}

	for {
		action, err := ui.ShowMainMenu()
		if err != nil {
			log.Fatal("Menu error:", err)
		}

		switch action {
		case ui.CreatePipeline:
			if err := createNewPipeline(cfg); err != nil {
				fmt.Printf("❌ Pipeline creation failed: %v\n", err)
				continue
			}
			fmt.Println("\n🎉 Pipeline created successfully!")
			fmt.Println("   Your Ubuntu-powered CI/CD is ready!")

		case ui.ViewPipelines:
			if err := ui.ViewExistingPipelines(cfg); err != nil {
				fmt.Printf("❌ Error viewing pipelines: %v\n", err)
			}

		case ui.ExitApp:
			fmt.Println("👋 Sawubona! Thanks for using Elephant TF CI!")
			os.Exit(0)
		}
		
		fmt.Println() // Add spacing between menu cycles
	}
}

func createNewPipeline(cfg *config.Config) error {
	// Step 1: Repository Selection
	fmt.Println("Step 1: Repository Selection")
	if err := ui.SelectRepositoryAndBranch(cfg); err != nil {
		return fmt.Errorf("repository selection failed: %w", err)
	}

	// Step 2: AWS Configuration  
	fmt.Println("\nStep 2: AWS Configuration")
	if err := ui.AWSConfigForm(cfg); err != nil {
		return fmt.Errorf("AWS configuration failed: %w", err)
	}

	// Step 3: Repository Settings
	fmt.Println("\nStep 3: Repository Settings")
	if err := ui.RepoConfigForm(cfg); err != nil {
		return fmt.Errorf("repository configuration failed: %w", err)
	}

	// Step 4: Confirmation
	fmt.Println("\nStep 4: Review & Confirm")
	confirm, err := ui.ConfirmationForm(cfg)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}

	if !confirm {
		fmt.Println("❌ Pipeline creation cancelled")
		return nil
	}

	// Step 4: Create Pipeline
	fmt.Println("\nStep 4: Creating Pipeline")
	pm := ui.NewPipelineManager(cfg)
	if err := pm.CreatePipeline(); err != nil {
		return fmt.Errorf("pipeline creation failed: %w", err)
	}

	return nil
}

func loadEmbeddedEnv() {
	lines := strings.Split(embeddedEnv, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}