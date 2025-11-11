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
const version = "v1.0.2"

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