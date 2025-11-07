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
	fmt.Println("⚙️  Workflow: .github/workflows/terraform.yml")
	fmt.Println("🔐 Secrets: Configured and encrypted")
	fmt.Println("🌍 Environments: dev, qa, prod")
	fmt.Println()
	fmt.Println("🚀 Next Steps:")
	fmt.Println("  1. Push your Terraform code to the repository")
	fmt.Println("  2. Create a pull request to trigger the pipeline")
	fmt.Println("  3. Monitor your deployments in GitHub Actions")
	fmt.Println()
	fmt.Println("💫 Your Ubuntu-powered CI/CD pipeline is ready!")

	return nil
}

