package ui

import (
	"context"
	"fmt"
	"indlovu-pipeline/internal/auth"
	"indlovu-pipeline/internal/config"
	"indlovu-pipeline/internal/pipeline"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/google/go-github/v56/github"
)

func GitHubConfigForm(cfg *config.Config) error {
	var proceed bool

	// Ask to sign in with GitHub
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Sign in with GitHub?").
				Description("🔐 Authorize this app to access your GitHub account").
				Affirmative("Sign in with GitHub").
				Negative("Cancel").
				Value(&proceed),
		).Title("🐙 GitHub OAuth").Description("Step 1 of 5: Authenticate with GitHub"),
	).Run()
	if err != nil {
		return err
	}

	if !proceed {
		return fmt.Errorf("authentication cancelled")
	}

	// Start OAuth flow
	fmt.Println("\n🔐 Starting GitHub OAuth flow...")
	oauth, err := auth.NewGitHubOAuth()
	if err != nil {
		return fmt.Errorf("OAuth setup failed: %w", err)
	}
	authURL := oauth.GetAuthURL()

	fmt.Println("🌐 Opening GitHub authorization page...")
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Please open this URL manually: %s\n", authURL)
	}

	fmt.Println("📝 Please sign in with your GitHub credentials in the browser")
	fmt.Println("⏳ Waiting for authorization...")

	// Start local server to receive callback
	if err := oauth.StartServer(); err != nil && err.Error() != "http: Server closed" {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	token := oauth.GetToken()
	if token == "" {
		return fmt.Errorf("no access token received")
	}

	cfg.GitHub.Token = token
	fmt.Println("\n🎉 GitHub OAuth authentication successful!")

	// Get authenticated user info
	pipelineCfg := &pipeline.Config{}
	setup := pipeline.NewPipelineSetup(cfg.GitHub.Token, pipelineCfg)
	user, err := setup.GetAuthenticatedUser()
	if err != nil {
		fmt.Printf("⚠️  Could not get user info: %v\n", err)
	} else if user != nil {
		if user.Login != nil {
			fmt.Printf("👤 Authenticated as: %s\n", *user.Login)
			cfg.GitHub.Username = *user.Login
		}
	}

	return selectRepository(cfg)
}

func selectRepository(cfg *config.Config) error {
	pipelineCfg := &pipeline.Config{}
	setup := pipeline.NewPipelineSetup(cfg.GitHub.Token, pipelineCfg)

	// Get user repos
	userRepos, err := setup.GetUserRepos()
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	// Get organizations
	orgs, _ := setup.GetUserOrgs()

	// Build repo options
	var repoOptions []huh.Option[string]
	for _, repo := range userRepos {
		repoName := fmt.Sprintf("%s/%s", *repo.Owner.Login, *repo.Name)
		repoOptions = append(repoOptions, huh.NewOption(repoName, repoName))
	}

	// Add org repos
	for _, org := range orgs {
		orgRepos, err := setup.GetOrgRepos(*org.Login)
		if err != nil {
			continue
		}
		for _, repo := range orgRepos {
			repoName := fmt.Sprintf("%s/%s", *repo.Owner.Login, *repo.Name)
			repoOptions = append(repoOptions, huh.NewOption(repoName, repoName))
		}
	}

	// Check if we have any repositories
	if len(repoOptions) == 0 {
		fmt.Println("❌ No repositories found. Please enter manually:")
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Repository Owner").
					Placeholder("username or organization").
					Value(&cfg.GitHub.Username),

				huh.NewInput().
					Title("Repository Name").
					Placeholder("my-terraform-project").
					Value(&cfg.GitHub.RepoName),
			),
		).Run()
	}



	var selectedRepo string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Repository").
				Description(fmt.Sprintf("📚 Choose from %d available repositories", len(repoOptions))).
				Options(repoOptions...).
				Value(&selectedRepo),
		).Title("📁 Repository Selection").Description("Step 2 of 5: Choose your target repository"),
	).Run()
	if err != nil {
		return err
	}

	// Parse owner/repo
	parts := strings.Split(selectedRepo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s", selectedRepo)
	}
	cfg.GitHub.Username = parts[0]
	cfg.GitHub.RepoName = parts[1]

	fmt.Printf("\n✅ Repository selected: %s/%s\n", cfg.GitHub.Username, cfg.GitHub.RepoName)

	// Get branches for selected repository
	return selectBranch(cfg, setup)
}

func selectBranch(cfg *config.Config, setup *pipeline.PipelineSetup) error {
	// Get branches
	branches, _, err := setup.GetClient().Repositories.ListBranches(context.Background(), cfg.GitHub.Username, cfg.GitHub.RepoName, &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		// If can't get branches, default to main
		cfg.GitHub.Branch = "main"
		return nil
	}

	// Build branch options
	var branchOptions []huh.Option[string]
	for _, branch := range branches {
		branchOptions = append(branchOptions, huh.NewOption(*branch.Name, *branch.Name))
	}

	// If no branches found, default to main
	if len(branchOptions) == 0 {
		cfg.GitHub.Branch = "main"
		return nil
	}

	var selectedBranch string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Branch").
				Description(fmt.Sprintf("🌱 Choose from %d available branches", len(branchOptions))).
				Options(branchOptions...).
				Value(&selectedBranch),
		).Title("🌳 Branch Selection").Description("Step 3 of 5: Choose your target branch"),
	).Run()
	if err != nil {
		return err
	}

	cfg.GitHub.Branch = selectedBranch
	fmt.Printf("✅ Branch selected: %s\n", cfg.GitHub.Branch)
	return nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func AWSConfigForm(cfg *config.Config) error {
	// Skip backend configuration - handled automatically
	return selectAWSRegion(cfg)
}



func selectAWSRegion(cfg *config.Config) error {
	// Build region options
	var regionOptions []huh.Option[string]
	regionOptions = append(regionOptions, huh.NewOption("US East 1 (Virginia)", "us-east-1"))
	regionOptions = append(regionOptions, huh.NewOption("US East 2 (Ohio)", "us-east-2"))
	regionOptions = append(regionOptions, huh.NewOption("AF South 1 (Cape Town)", "af-south-1"))
	regionOptions = append(regionOptions, huh.NewOption("EU West 1 (Ireland)", "eu-west-1"))
	regionOptions = append(regionOptions, huh.NewOption("EU Central 1 (Frankfurt)", "eu-central-1"))
	regionOptions = append(regionOptions, huh.NewOption("✏️ Enter Custom Region", "custom"))

	var selectedRegion string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Region").
				Description("🌍 Choose from popular regions or enter custom").
				Options(regionOptions...).
				Value(&selectedRegion),
		).Title("🌍 Region Selection").Description("Step 2: Choose your AWS region"),
	).Run()
	if err != nil {
		return err
	}

	// Handle custom region input
	if selectedRegion == "custom" {
		var customRegion string
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Enter AWS Region").
					Description("🌍 Enter any valid AWS region (e.g., ap-southeast-1, ca-central-1)").
					Placeholder("us-west-2").
					Validate(func(s string) error {
						if len(s) == 0 {
							return fmt.Errorf("region is required")
						}
						return nil
					}).
					Value(&customRegion),
			).Title("✏️ Custom Region").Description("Enter your preferred AWS region"),
		).Run()
		if err != nil {
			return err
		}
		cfg.AWS.Region = customRegion
	} else {
		cfg.AWS.Region = selectedRegion
	}

	fmt.Printf("✅ Region selected: %s\n", cfg.AWS.Region)

	// Step 3: AWS Role ARNs
	return AWSRoleConfigForm(cfg)
}

func AWSRoleConfigForm(cfg *config.Config) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Pipeline Role ARN").
				Description("🔐 AWS IAM Role ARN for GitHub Actions pipeline").
				Placeholder("arn:aws:iam::123456789012:role/GitHubActions-Pipeline").
				Validate(func(s string) error {
					if len(s) == 0 {
						return fmt.Errorf("pipeline role ARN is required")
					}
					return nil
				}).
				Value(&cfg.AWS.PipelineRoleARN),

			huh.NewConfirm().
				Title("Enable Security Scanning?").
				Description("🔒 Fail pipeline on security/linting issues (Checkov, TFLint, TFSec)").
				Affirmative("Yes, fail on issues").
				Negative("No, continue on issues").
				Value(&cfg.AWS.FailOnSecurityIssues),
		).Title("🔐 Pipeline Configuration").Description("Step 3: Configure pipeline settings"),
	).Run()
}

func RepoConfigForm(cfg *config.Config) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Repository Description").
				Placeholder("Terraform infrastructure with CI/CD pipeline").
				Value(&cfg.Repo.Description),

			huh.NewConfirm().
				Title("Private Repository?").
				Value(&cfg.Repo.Private),
		).Title("📁 Repository Settings"),
	)

	return form.Run()
}

func ConfirmationForm(cfg *config.Config) (bool, error) {
	var confirm bool

	summary := fmt.Sprintf(`
🚀 Elephant TF CI Pipeline Setup Summary

🐙 GitHub Configuration:
  • Username: %s
  • Repository: %s
  • Branch: %s
  • Private: %t

☁️ AWS Configuration:
  • Region: %s
  • Pipeline Role: %s
  • Backend: Auto-managed S3 bucket

🚀 Pipeline Features:
  • Terraform CI/CD workflow
  • Security scanning (Checkov, TFSec, TFLint)
  • Multi-environment support
  • OIDC keyless authentication
  • Encrypted GitHub secrets
  • Automatic S3 backend creation

⚡ This will create/update:
  1. GitHub repository (if needed)
  2. S3 backend configuration (auto-managed)
  3. .github/workflows/terraform.yml
  4. GitHub repository secrets

Ready to proceed?`, 
		cfg.GitHub.Username,
		cfg.GitHub.RepoName,
		cfg.GitHub.Branch,
		cfg.Repo.Private,
		cfg.AWS.Region,
		cfg.AWS.PipelineRoleARN,
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Review Configuration").
				Description(summary),

			huh.NewConfirm().
				Title("Create Pipeline?").
				Description("💫 Final step: Confirm to create your Terraform CI/CD pipeline").
				Affirmative("Yes, Create Pipeline!").
				Negative("Cancel").
				Value(&confirm),
		).Title("🔍 Final Review").Description("Step 4 of 4: Review and confirm your configuration"),
	)

	err := form.Run()
	return confirm, err
}