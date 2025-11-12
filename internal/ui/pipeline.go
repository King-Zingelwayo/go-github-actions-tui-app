package ui

import (
	"fmt"
	"indlovu-pipeline/internal/config"
	"indlovu-pipeline/internal/pipeline"
)

type PipelineManager struct {
	config *config.Config
}

func NewPipelineManager(cfg *config.Config) *PipelineManager {
	return &PipelineManager{
		config: cfg,
	}
}

func (pm *PipelineManager) CreatePipeline() error {
	pipelineCfg := &pipeline.Config{
		Owner:                pm.config.GitHub.Username,
		Repo:                 pm.config.GitHub.RepoName,
		Branch:               pm.config.GitHub.Branch,
		AWSRegion:            pm.config.AWS.Region,
		TFStateBucket:        pm.config.AWS.StateBucket,
		PipelineRoleARN:      pm.config.AWS.PipelineRoleARN,
		FailOnSecurityIssues: pm.config.AWS.FailOnSecurityIssues,
		HasExistingBackend:   pm.config.AWS.HasExistingBackend,
	}

	setup := pipeline.NewPipelineSetup(pm.config.GitHub.Token, pipelineCfg)

	fmt.Println("🚀 Creating your Indlovu Pipeline...")
	fmt.Println("🔄 This may take a few moments...")
	
	if err := setup.SetupPipeline(); err != nil {
		return fmt.Errorf("failed to setup pipeline: %w", err)
	}

	fmt.Println("\n✨ ================================= ✨")
	fmt.Println("🎉 SUCCESS! Indlovu Pipeline Created!")
	fmt.Println("✨ ================================= ✨")
	fmt.Println()
	fmt.Printf("🔗 Repository: https://github.com/%s/%s\n", 
		pm.config.GitHub.Username, pm.config.GitHub.RepoName)
	fmt.Printf("🌳 Branch: %s\n", pm.config.GitHub.Branch)
	fmt.Println("⚙️  Workflow:")
	fmt.Println("   • .github/workflows/terraform.yml (CI/CD)")
	
	if !pm.config.AWS.HasExistingBackend {
		fmt.Println("🪣 Backend:")
		fmt.Printf("   • backend.tf (S3 bucket: %s)\n", pipelineCfg.TFStateBucket)
	}
	
	fmt.Println("🔐 Secrets: Configured and encrypted")
	fmt.Println()
	fmt.Println("🚀 Next Steps:")
	fmt.Println("  1. Push your Terraform code to the repository")
	fmt.Println("  2. Create a pull request to trigger the pipeline")
	fmt.Println("  3. Monitor your deployments in GitHub Actions")
	fmt.Println("  4. Use 'View Existing Pipelines' to manage resources")
	if !pm.config.AWS.HasExistingBackend {
		fmt.Println("  5. Create the S3 bucket in AWS before running Terraform")
	}
	fmt.Println()
	fmt.Println("💫 Your Ubuntu-powered CI/CD pipeline is ready!")

	// Open GitHub Actions page
	actionsURL := fmt.Sprintf("https://github.com/%s/%s/actions", 
		pm.config.GitHub.Username, pm.config.GitHub.RepoName)
	fmt.Printf("\n🌐 Opening GitHub Actions: %s\n", actionsURL)
	if err := openBrowser(actionsURL); err != nil {
		fmt.Printf("Please open this URL manually: %s\n", actionsURL)
	}

	return nil
}