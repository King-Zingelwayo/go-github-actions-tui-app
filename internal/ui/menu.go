package ui

import (
	"context"
	"fmt"
	"indlovu-pipeline/internal/auth"
	"indlovu-pipeline/internal/config"
	"indlovu-pipeline/internal/pipeline"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/google/go-github/v56/github"
)

type MenuAction string

const (
	CreatePipeline MenuAction = "create"
	ViewPipelines  MenuAction = "view"
	ExitApp        MenuAction = "exit"
)

type PipelineAction string

const (
	DestroyResources PipelineAction = "destroy"
	OpenRepository   PipelineAction = "open_repo"
	OpenActions      PipelineAction = "open_actions"
	RefreshStatus    PipelineAction = "refresh"
	BackToMenu       PipelineAction = "back"
)

func ShowMainMenu() (MenuAction, error) {
	var action MenuAction

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[MenuAction]().
				Title("🐘 Elephant TF CI - Main Menu").
				Description("What would you like to do?").
				Options(
					huh.NewOption("🚀 Create New Pipeline", CreatePipeline),
					huh.NewOption("📋 View Existing Pipelines", ViewPipelines),
					huh.NewOption("🚪 Exit", ExitApp),
				).
				Value(&action),
		).Title("🌍 Ubuntu-powered Terraform CI/CD").Description("Choose your action"),
	).Run()

	return action, err
}

func ViewExistingPipelines(cfg *config.Config) error {
	// Get repositories with workflows
	repos, err := getRepositoriesWithWorkflows(cfg)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		fmt.Println("📭 No repositories with Terraform workflows found")
		return nil
	}

	// Show repository selection
	selectedRepo, err := selectRepositoryForViewing(repos)
	if err != nil {
		return err
	}

	if selectedRepo == "" {
		return nil // User cancelled
	}

	// Show pipeline status and actions
	return showPipelineStatus(cfg, selectedRepo)
}

func AuthenticateGitHub(cfg *config.Config) error {
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
		).Title("🐙 GitHub OAuth").Description("Authentication required"),
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

	return nil
}

func SelectRepositoryAndBranch(cfg *config.Config) error {
	return selectRepository(cfg)
}

func getRepositoriesWithWorkflows(cfg *config.Config) ([]string, error) {
	pipelineCfg := &pipeline.Config{}
	setup := pipeline.NewPipelineSetup(cfg.GitHub.Token, pipelineCfg)

	// Get user repos
	userRepos, err := setup.GetUserRepos()
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	// Get organizations
	orgs, _ := setup.GetUserOrgs()

	var reposWithWorkflows []string

	// Check user repos for workflows
	for _, repo := range userRepos {
		if hasWorkflow(setup.GetClient(), *repo.Owner.Login, *repo.Name) {
			repoName := fmt.Sprintf("%s/%s", *repo.Owner.Login, *repo.Name)
			reposWithWorkflows = append(reposWithWorkflows, repoName)
		}
	}

	// Check org repos for workflows
	for _, org := range orgs {
		orgRepos, err := setup.GetOrgRepos(*org.Login)
		if err != nil {
			continue
		}
		for _, repo := range orgRepos {
			if hasWorkflow(setup.GetClient(), *repo.Owner.Login, *repo.Name) {
				repoName := fmt.Sprintf("%s/%s", *repo.Owner.Login, *repo.Name)
				reposWithWorkflows = append(reposWithWorkflows, repoName)
			}
		}
	}

	return reposWithWorkflows, nil
}

func hasWorkflow(client *github.Client, owner, repo string) bool {
	// Check for terraform.yml workflow
	_, _, _, err1 := client.Repositories.GetContents(
		context.Background(),
		owner,
		repo,
		".github/workflows/terraform.yml",
		&github.RepositoryContentGetOptions{},
	)
	
	// Check for destroy.yml workflow
	_, _, _, err2 := client.Repositories.GetContents(
		context.Background(),
		owner,
		repo,
		".github/workflows/destroy.yml",
		&github.RepositoryContentGetOptions{},
	)
	
	// Return true if either workflow exists
	return err1 == nil || err2 == nil
}

func selectRepositoryForViewing(repos []string) (string, error) {
	if len(repos) == 0 {
		return "", nil
	}

	var repoOptions []huh.Option[string]
	for _, repo := range repos {
		repoOptions = append(repoOptions, huh.NewOption(repo, repo))
	}
	repoOptions = append(repoOptions, huh.NewOption("← Back to Main Menu", ""))

	var selectedRepo string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Repository").
				Description(fmt.Sprintf("📚 Found %d repositories with Terraform workflows", len(repos))).
				Options(repoOptions...).
				Value(&selectedRepo),
		).Title("📋 Existing Pipelines").Description("Choose a repository to manage"),
	).Run()

	return selectedRepo, err
}

func showPipelineStatus(cfg *config.Config, repoName string) error {
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s", repoName)
	}
	owner, repo := parts[0], parts[1]

	pipelineCfg := &pipeline.Config{}
	setup := pipeline.NewPipelineSetup(cfg.GitHub.Token, pipelineCfg)

	// Get recent workflow runs
	runs, err := getRecentWorkflowRuns(setup.GetClient(), owner, repo)
	if err != nil {
		fmt.Printf("⚠️ Could not fetch workflow runs: %v\n", err)
	}

	// Display pipeline status
	statusInfo := fmt.Sprintf(`
📊 Pipeline Status for %s

🔗 Repository: https://github.com/%s/%s
📋 Actions: https://github.com/%s/%s/actions

🚀 Recent Workflow Runs:
%s

⚙️ Available Workflows:
  • terraform.yml (CI/CD Pipeline)
  • destroy.yml (Resource Cleanup)
`,
		repoName, owner, repo, owner, repo, formatWorkflowRuns(runs))

	var action PipelineAction
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Pipeline Overview").
				Description(statusInfo),
			huh.NewSelect[PipelineAction]().
				Title("Pipeline Management Options").
				Description("Choose an action to perform on this pipeline").
				Options(
					huh.NewOption("🌐 Open Repository in Browser", OpenRepository),
					huh.NewOption("📋 Open GitHub Actions", OpenActions),
					huh.NewOption("🔄 Refresh Pipeline Status", RefreshStatus),
					huh.NewOption("💥 Destroy Environment Resources", DestroyResources),
					huh.NewOption("← Back to Repository List", BackToMenu),
				).
				Value(&action),
		).Title("🛠️ Pipeline Management"),
	).Run()
	if err != nil {
		return err
	}

	switch action {
	case OpenRepository:
		repoURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)
		fmt.Printf("🌐 Opening repository: %s\n", repoURL)
		if err := openBrowser(repoURL); err != nil {
			fmt.Printf("Please open this URL manually: %s\n", repoURL)
		}
		return showPipelineStatus(cfg, repoName)
		
	case OpenActions:
		actionsURL := fmt.Sprintf("https://github.com/%s/%s/actions", owner, repo)
		fmt.Printf("📋 Opening GitHub Actions: %s\n", actionsURL)
		if err := openBrowser(actionsURL); err != nil {
			fmt.Printf("Please open this URL manually: %s\n", actionsURL)
		}
		return showPipelineStatus(cfg, repoName)
		
	case RefreshStatus:
		fmt.Println("🔄 Refreshing pipeline status...")
		return showPipelineStatus(cfg, repoName)
		
	case DestroyResources:
		return handleResourceDestruction(cfg, owner, repo)
		
	case BackToMenu:
		return ViewExistingPipelines(cfg)
	}

	return nil
}



func confirmAndDestroyResources(cfg *config.Config, owner, repo string) error {
	// Get repository branches to determine available environments
	pipelineCfg := &pipeline.Config{}
	setup := pipeline.NewPipelineSetup(cfg.GitHub.Token, pipelineCfg)
	
	branches, _, err := setup.GetClient().Repositories.ListBranches(context.Background(), owner, repo, &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return fmt.Errorf("failed to get repository branches: %w", err)
	}

	// Build environment options with smart detection - only for branches with workflows
	var envOptions []huh.Option[string]
	
	for _, branch := range branches {
		branchName := *branch.Name
		
		// Check if branch has .github/workflows folder
		if !hasWorkflowsFolder(setup.GetClient(), owner, repo, branchName) {
			continue
		}
		
		// Check for tfvars file and environment variable
		envName := detectEnvironmentFromTfvars(setup.GetClient(), owner, repo, branchName)
		if envName != "" {
			// Found environment in tfvars
			envOptions = append(envOptions, huh.NewOption(fmt.Sprintf("Environment: %s (branch: %s)", envName, branchName), envName))
		} else {
			// No tfvars or no environment variable, use branch name
			envOptions = append(envOptions, huh.NewOption(fmt.Sprintf("Branch: %s", branchName), branchName))
		}
	}
	
	if len(envOptions) == 0 {
		return fmt.Errorf("no branches found in repository")
	}

	// Let user select which environment/branch to destroy
	var selectedEnvOrBranch string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Environment to Destroy").
				Description(fmt.Sprintf("🌍 Choose from %d available environments/branches", len(envOptions))).
				Options(envOptions...).
				Value(&selectedEnvOrBranch),
		).Title("🌍 Environment Selection").Description("Select the environment/branch whose resources you want to destroy"),
	).Run()
	if err != nil {
		return err
	}

	// Find the corresponding branch for the selected environment
	var targetBranch string
	for _, branch := range branches {
		branchName := *branch.Name
		
		// Skip branches without workflows
		if !hasWorkflowsFolder(setup.GetClient(), owner, repo, branchName) {
			continue
		}
		
		envName := detectEnvironmentFromTfvars(setup.GetClient(), owner, repo, branchName)
		if (envName != "" && envName == selectedEnvOrBranch) || (envName == "" && branchName == selectedEnvOrBranch) {
			targetBranch = branchName
			break
		}
	}
	
	if targetBranch == "" {
		return fmt.Errorf("could not find branch for selected environment: %s", selectedEnvOrBranch)
	}

	// First confirmation
	var firstConfirm bool
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("⚠️ DANGER: Destroy %s Resources?", selectedEnvOrBranch)).
				Description(fmt.Sprintf("This will run 'terraform destroy' on %s (branch: %s) for %s/%s.\nThis action CANNOT be undone!", selectedEnvOrBranch, targetBranch, owner, repo)).
				Affirmative("Yes, I understand the risks").
				Negative("Cancel").
				Value(&firstConfirm),
		).Title("💥 Resource Destruction Warning"),
	).Run()
	if err != nil || !firstConfirm {
		return err
	}

	// Second confirmation with typing
	var confirmText string
	expectedText := fmt.Sprintf("%s/%s-%s", owner, repo, selectedEnvOrBranch)
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Type to confirm destruction").
				Description(fmt.Sprintf("Type '%s' to confirm %s destruction:", expectedText, selectedEnvOrBranch)).
				Placeholder(expectedText).
				Validate(func(s string) error {
					if s != expectedText {
						return fmt.Errorf("must match exactly: %s", expectedText)
					}
					return nil
				}).
				Value(&confirmText),
		).Title("🔒 Confirmation"),
	).Run()
	if err != nil {
		return err
	}

	// Third and final confirmation
	var finalConfirm bool
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("🚨 FINAL WARNING").
				Description(fmt.Sprintf("This will PERMANENTLY DESTROY all %s resources.\nAre you absolutely sure?", selectedEnvOrBranch)).
				Affirmative(fmt.Sprintf("YES, DESTROY %s", strings.ToUpper(selectedEnvOrBranch))).
				Negative("Cancel").
				Value(&finalConfirm),
		).Title("💀 Point of No Return"),
	).Run()
	if err != nil || !finalConfirm {
		return err
	}

	return executeDestroy(cfg, owner, repo, targetBranch, selectedEnvOrBranch)
}

func executeDestroy(cfg *config.Config, owner, repo, branchName, envName string) error {
	// Read AWS configuration from backend.tf
	pipelineCfg := &pipeline.Config{Owner: owner, Repo: repo}
	setup := pipeline.NewPipelineSetup(cfg.GitHub.Token, pipelineCfg)
	
	// Update destroy workflow with latest template on default branch (required for dispatch)
	fmt.Println("🛠️ Updating destroy workflow with latest template...")
	
	// Get repository info to find default branch
	repoInfo, _, err := setup.GetClient().Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get repository info: %w", err)
	}
	
	defaultBranch := "main"
	if repoInfo.DefaultBranch != nil {
		defaultBranch = *repoInfo.DefaultBranch
	}
	
	if err := setup.CreateDestroyWorkflowFileOnBranch(defaultBranch); err != nil {
		return fmt.Errorf("failed to update destroy workflow: %w", err)
	}
	fmt.Println("✅ Destroy workflow updated successfully")
	
	// Wait for GitHub to process the workflow update
	fmt.Println("⏳ Waiting for GitHub to process workflow update...")
	time.Sleep(5 * time.Second)
	
	stateBucket, awsRegion, err := setup.GetBackendConfig()
	if err != nil {
		return fmt.Errorf("failed to read backend configuration: %w", err)
	}
	
	fmt.Printf("📍 Read from backend.tf - Region: %s, Bucket: %s\n", awsRegion, stateBucket)

	fmt.Println("🚀 Triggering Terraform Destroy workflow...")
	fmt.Printf("📍 Repository: %s/%s\n", owner, repo)
	fmt.Printf("🌳 Branch: %s\n", branchName)
	fmt.Printf("🌍 Environment: %s\n", envName)

	// Use the selected branch name directly
	fmt.Printf("🌳 Using branch: %s\n", branchName)

	// Trigger destroy workflow dispatch
	_, err = setup.GetClient().Actions.CreateWorkflowDispatchEventByFileName(
		context.Background(),
		owner,
		repo,
		"destroy.yml",
		github.CreateWorkflowDispatchEventRequest{
			Ref: defaultBranch,
			Inputs: map[string]interface{}{
				"environment":     branchName,
				"aws_region":      awsRegion,
				"tf_state_bucket": stateBucket,
				"confirm_destroy": "DESTROY",
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to trigger destroy workflow: %w", err)
	}

	fmt.Println("✅ Destroy workflow triggered successfully!")
	fmt.Printf("🔗 Monitor progress: https://github.com/%s/%s/actions\n", owner, repo)
	fmt.Println("📍 The destroy workflow will:")
	fmt.Println("   1. Plan the destruction")
	fmt.Println("   2. Wait 10 seconds for final confirmation")
	fmt.Println("   3. Execute terraform destroy")
	fmt.Println("⚠️  This action cannot be undone!")
	
	// Open GitHub Actions page to monitor the destroy workflow
	actionsURL := fmt.Sprintf("https://github.com/%s/%s/actions", owner, repo)
	fmt.Printf("\n🌐 Opening GitHub Actions to monitor destroy workflow: %s\n", actionsURL)
	if err := openBrowser(actionsURL); err != nil {
		fmt.Printf("Please open this URL manually: %s\n", actionsURL)
	}
	
	return nil
}

func getStringValue(s *string) string {
	if s == nil {
		return "N/A"
	}
	return *s
}

func getBoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func hasWorkflowsFolder(client *github.Client, owner, repo, branch string) bool {
	_, _, _, err := client.Repositories.GetContents(
		context.Background(),
		owner,
		repo,
		".github/workflows",
		&github.RepositoryContentGetOptions{Ref: branch},
	)
	return err == nil
}

func detectEnvironmentFromTfvars(client *github.Client, owner, repo, branch string) string {
	// Look for common tfvars files
	tfvarsFiles := []string{
		"terraform.tfvars",
		"variables.tfvars",
		fmt.Sprintf("%s.tfvars", branch),
		"env.tfvars",
	}
	
	for _, filename := range tfvarsFiles {
		fileContent, _, _, err := client.Repositories.GetContents(
			context.Background(),
			owner,
			repo,
			filename,
			&github.RepositoryContentGetOptions{Ref: branch},
		)
		if err != nil {
			continue // File doesn't exist, try next
		}
		
		// Decode file content
		content, err := fileContent.GetContent()
		if err != nil {
			continue
		}
		
		// Look for environment variable patterns
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") {
				continue // Skip comments
			}
			
			// Look for environment = "value" patterns
			if strings.Contains(line, "environment") && strings.Contains(line, "=") {
				parts := strings.Split(line, "=")
				if len(parts) >= 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					
					// Remove quotes and extract environment name
					if key == "environment" || strings.Contains(key, "env") {
						value = strings.Trim(value, `"'`)
						if value != "" {
							return value
						}
					}
				}
			}
		}
	}
	
	return "" // No environment found
}

func getRecentWorkflowRuns(client *github.Client, owner, repo string) ([]*github.WorkflowRun, error) {
	runs, _, err := client.Actions.ListRepositoryWorkflowRuns(context.Background(), owner, repo, &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 5},
	})
	if err != nil {
		return nil, err
	}
	return runs.WorkflowRuns, nil
}

func formatWorkflowRuns(runs []*github.WorkflowRun) string {
	if len(runs) == 0 {
		return "  No recent runs found"
	}

	var result strings.Builder
	for i, run := range runs {
		if i >= 5 { // Limit to 5 most recent
			break
		}
		status := "🔄" // Default
		switch *run.Status {
		case "completed":
			if *run.Conclusion == "success" {
				status = "✅"
			} else {
				status = "❌"
			}
		case "in_progress":
			status = "🔄"
		case "queued":
			status = "⏳"
		}
		result.WriteString(fmt.Sprintf("  %s %s - %s (%s)\n    🔗 %s\n", 
			status, 
			getStringValue(run.Name),
			run.CreatedAt.Format("Jan 02 15:04"),
			getStringValue(run.HeadBranch),
			getStringValue(run.HTMLURL)))
	}
	return result.String()
}

func handleResourceDestruction(cfg *config.Config, owner, repo string) error {
	return confirmAndDestroyResources(cfg, owner, repo)
}

func ensureDestroyWorkflow(cfg *config.Config, owner, repo string) error {
	return nil
}