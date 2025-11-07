package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v56/github"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/oauth2"
)

type PipelineSetup struct {
	client *github.Client
	config *Config
}

type Config struct {
	GitHubToken           string
	Owner                 string
	Repo                  string
	Branch                string
	AWSRegion             string
	TFStateBucket         string
	PipelineRoleARN       string
	FailOnSecurityIssues  bool
}

func NewPipelineSetup(token string, cfg *Config) *PipelineSetup {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	cfg.GitHubToken = token
	return &PipelineSetup{
		client: client,
		config: cfg,
	}
}

func (p *PipelineSetup) GetAuthenticatedUser() (*github.User, error) {
	user, _, err := p.client.Users.Get(context.Background(), "")
	return user, err
}

func (p *PipelineSetup) GetUserRepos() ([]*github.Repository, error) {
	repos, _, err := p.client.Repositories.List(context.Background(), "", &github.RepositoryListOptions{
		Visibility: "all",
		Affiliation: "owner,collaborator,organization_member",
		Sort: "updated",
		ListOptions: github.ListOptions{PerPage: 100},
	})
	return repos, err
}

func (p *PipelineSetup) GetOrgRepos(org string) ([]*github.Repository, error) {
	repos, _, err := p.client.Repositories.ListByOrg(context.Background(), org, &github.RepositoryListByOrgOptions{
		Type: "all",
		ListOptions: github.ListOptions{PerPage: 100},
	})
	return repos, err
}

func (p *PipelineSetup) GetUserOrgs() ([]*github.Organization, error) {
	orgs, _, err := p.client.Organizations.List(context.Background(), "", &github.ListOptions{PerPage: 100})
	return orgs, err
}

func (p *PipelineSetup) GetClient() *github.Client {
	return p.client
}

func (p *PipelineSetup) SetupPipeline() error {
	// Check if repository exists, create if not
	if err := p.ensureRepository(); err != nil {
		return fmt.Errorf("failed to ensure repository: %w", err)
	}

	if err := p.createWorkflowFile(); err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	if err := p.setupSecrets(); err != nil {
		return fmt.Errorf("failed to setup secrets: %w", err)
	}

	if err := p.setupEnvironments(); err != nil {
		return fmt.Errorf("failed to setup environments: %w", err)
	}

	fmt.Println("✅ Pipeline setup completed successfully!")
	return nil
}

func encryptSecret(plaintext, publicKeyB64 string) (string, error) {
	// Decode the public key
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return "", err
	}

	// Convert to NaCl public key format
	var publicKey [32]byte
	copy(publicKey[:], publicKeyBytes)

	// Encrypt the plaintext
	encrypted, err := box.SealAnonymous(nil, []byte(plaintext), &publicKey, rand.Reader)
	if err != nil {
		return "", err
	}

	// Return base64 encoded encrypted value
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func (p *PipelineSetup) createWorkflowFile() error {
	workflowPath := ".github/workflows/terraform.yml"
	templatePath := filepath.Join("internal", "templates", "deployment.yml")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}

	// Replace SELECTED_BRANCH placeholder with actual branch
	branch := p.config.Branch
	if branch == "" {
		branch = "main"
	}
	contentStr := string(content)
	contentStr = strings.ReplaceAll(contentStr, "SELECTED_BRANCH", branch)
	content = []byte(contentStr)

	// Check if file exists
	existingFile, _, _, err := p.client.Repositories.GetContents(
		context.Background(),
		p.config.Owner,
		p.config.Repo,
		workflowPath,
		&github.RepositoryContentGetOptions{
			Ref: branch,
		},
	)

	fileOptions := &github.RepositoryContentFileOptions{
		Message: github.String("Add/Update Terraform CI/CD pipeline"),
		Content: content,
		Branch:  github.String(branch),
	}

	if err == nil && existingFile != nil {
		// File exists, update it
		fileOptions.SHA = existingFile.SHA
		_, _, err = p.client.Repositories.UpdateFile(
			context.Background(),
			p.config.Owner,
			p.config.Repo,
			workflowPath,
			fileOptions,
		)
	} else {
		// File doesn't exist, create it
		_, _, err = p.client.Repositories.CreateFile(
			context.Background(),
			p.config.Owner,
			p.config.Repo,
			workflowPath,
			fileOptions,
		)
	}

	return err
}

func (p *PipelineSetup) setupSecrets() error {
	secrets := map[string]string{
		"AWS_REGION":        p.config.AWSRegion,
		"TF_STATE_BUCKET":   p.config.TFStateBucket,
		"PIPELINE_ROLE_ARN": p.config.PipelineRoleARN,
	}

	for name, value := range secrets {
		if value == "" {
			continue
		}
		
		// Get public key for encryption
		pubKey, _, err := p.client.Actions.GetRepoPublicKey(context.Background(), p.config.Owner, p.config.Repo)
		if err != nil {
			return err
		}

		// Encrypt the secret value
		encryptedValue, err := encryptSecret(value, pubKey.GetKey())
		if err != nil {
			return err
		}

		_, err = p.client.Actions.CreateOrUpdateRepoSecret(
			context.Background(),
			p.config.Owner,
			p.config.Repo,
			&github.EncryptedSecret{
				Name:           name,
				EncryptedValue: encryptedValue,
				KeyID:          pubKey.GetKeyID(),
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PipelineSetup) ensureRepository() error {
	// Check if repository exists
	_, _, err := p.client.Repositories.Get(context.Background(), p.config.Owner, p.config.Repo)
	if err == nil {
		fmt.Printf("✅ Repository %s/%s already exists\n", p.config.Owner, p.config.Repo)
		return nil
	}

	// Create repository if it doesn't exist
	fmt.Printf("📁 Creating repository %s/%s...\n", p.config.Owner, p.config.Repo)
	repo := &github.Repository{
		Name:        github.String(p.config.Repo),
		Description: github.String("Terraform infrastructure with CI/CD pipeline"),
		Private:     github.Bool(false),
	}

	_, _, err = p.client.Repositories.Create(context.Background(), "", repo)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	fmt.Printf("✅ Repository %s/%s created successfully\n", p.config.Owner, p.config.Repo)
	return nil
}

func (p *PipelineSetup) setupEnvironments() error {
	// Note: Environment creation requires GitHub Enterprise or specific API access
	// For now, we'll skip this step as it's not available in standard GitHub API
	fmt.Println("⚠️  Environment setup skipped - requires manual setup in GitHub repository settings")
	return nil
}