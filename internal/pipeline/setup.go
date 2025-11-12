package pipeline

import (
	_ "embed"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	mathrand "math/rand"
	"strings"
	"time"

	"github.com/google/go-github/v56/github"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/oauth2"
)

//go:embed templates/terraform-cicd.yml
var terraformCICDTemplate string

//go:embed templates/terraform-destroy.yml
var terraformDestroyTemplate string

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
	HasExistingBackend    bool
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
	if err := p.ensureRepository(); err != nil {
		return fmt.Errorf("failed to ensure repository: %w", err)
	}

	// Check if pipeline already exists and reuse bucket
	if p.config.TFStateBucket == "" {
		existingBucket, err := p.getExistingBucket()
		if err == nil && existingBucket != "" {
			// Found backend.tf with bucket name, but need to verify bucket exists
			fmt.Printf("🔍 Found existing backend.tf with bucket: %s\n", existingBucket)
			p.config.TFStateBucket = existingBucket
			p.config.HasExistingBackend = true
			fmt.Printf("♾️ Will reuse existing state bucket: %s\n", p.config.TFStateBucket)
		} else {
			p.config.TFStateBucket = p.generateBucketName()
			fmt.Printf("🪣 Generated new state bucket name: %s\n", p.config.TFStateBucket)
		}
	}

	if err := p.createPipelineFiles(); err != nil {
		return fmt.Errorf("failed to create pipeline files: %w", err)
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

func (p *PipelineSetup) generateBucketName() string {
	mathrand.Seed(time.Now().UnixNano())
	suffix := mathrand.Intn(99999)
	// Clean and truncate owner/repo names for valid S3 bucket names
	owner := p.cleanBucketNamePart(p.config.Owner, 10)
	repo := p.cleanBucketNamePart(p.config.Repo, 15)
	return fmt.Sprintf("tf-state-%s-%s-%05d", owner, repo, suffix)
}

func (p *PipelineSetup) cleanBucketNamePart(name string, maxLen int) string {
	// Convert to lowercase
	name = strings.ToLower(name)
	// Replace invalid characters with hyphens
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	// Remove any non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	name = result.String()
	// Truncate if too long
	if len(name) > maxLen {
		name = name[:maxLen]
	}
	// Remove trailing hyphens
	name = strings.TrimRight(name, "-")
	return name
}

func (p *PipelineSetup) createPipelineFiles() error {
	branch := p.config.Branch
	if branch == "" {
		branch = "main"
	}

	filesToUpdate := make(map[string][]byte)

	// Always create/update backend.tf with branch-based state path
	backendContent := fmt.Sprintf(`terraform {
  backend "s3" {
    bucket = "%s"
    key    = "%s/terraform.tfstate"
    region = "%s"
  }
}`, p.config.TFStateBucket, branch, p.config.AWSRegion)
	filesToUpdate["backend.tf"] = []byte(backendContent)
	fmt.Printf("📝 Preparing backend configuration with bucket: %s, key: %s/terraform.tfstate\n", p.config.TFStateBucket, branch)

	workflowContent := strings.ReplaceAll(terraformCICDTemplate, "SELECTED_BRANCH", branch)
	filesToUpdate[".github/workflows/terraform.yml"] = []byte(workflowContent)
	fmt.Println("📝 Preparing workflow configuration")

	return p.updateMultipleFiles(filesToUpdate, "Add/Update Terraform pipeline configuration", branch)
}

func (p *PipelineSetup) updateMultipleFiles(files map[string][]byte, commitMessage, branch string) error {
	ctx := context.Background()

	// Try to get existing branch reference
	ref, _, err := p.client.Git.GetRef(ctx, p.config.Owner, p.config.Repo, "refs/heads/"+branch)
	if err != nil {
		// Branch doesn't exist, create initial commit
		return p.createInitialCommit(files, commitMessage, branch)
	}

	commit, _, err := p.client.Git.GetCommit(ctx, p.config.Owner, p.config.Repo, *ref.Object.SHA)
	if err != nil {
		return fmt.Errorf("failed to get commit: %w", err)
	}

	baseTree, _, err := p.client.Git.GetTree(ctx, p.config.Owner, p.config.Repo, *commit.Tree.SHA, true)
	if err != nil {
		return fmt.Errorf("failed to get tree: %w", err)
	}

	var entries []*github.TreeEntry
	for path, content := range files {
		blob, _, err := p.client.Git.CreateBlob(ctx, p.config.Owner, p.config.Repo, &github.Blob{
			Content:  github.String(string(content)),
			Encoding: github.String("utf-8"),
		})
		if err != nil {
			return fmt.Errorf("failed to create blob for %s: %w", path, err)
		}

		entries = append(entries, &github.TreeEntry{
			Path: github.String(path),
			Mode: github.String("100644"),
			Type: github.String("blob"),
			SHA:  blob.SHA,
		})
	}

	newTree, _, err := p.client.Git.CreateTree(ctx, p.config.Owner, p.config.Repo, *baseTree.SHA, entries)
	if err != nil {
		return fmt.Errorf("failed to create tree: %w", err)
	}

	newCommit, _, err := p.client.Git.CreateCommit(ctx, p.config.Owner, p.config.Repo, &github.Commit{
		Message: github.String(commitMessage),
		Tree:    newTree,
		Parents: []*github.Commit{commit},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	_, _, err = p.client.Git.UpdateRef(ctx, p.config.Owner, p.config.Repo, &github.Reference{
		Ref: github.String("refs/heads/" + branch),
		Object: &github.GitObject{
			SHA: newCommit.SHA,
		},
	}, false)
	if err != nil {
		return fmt.Errorf("failed to update branch: %w", err)
	}

	fmt.Printf("✅ Updated %d files in single commit\n", len(files))
	return nil
}

func (p *PipelineSetup) createInitialCommit(files map[string][]byte, commitMessage, branch string) error {
	ctx := context.Background()

	// Create tree entries for files
	var entries []*github.TreeEntry
	for path, content := range files {
		blob, _, err := p.client.Git.CreateBlob(ctx, p.config.Owner, p.config.Repo, &github.Blob{
			Content:  github.String(string(content)),
			Encoding: github.String("utf-8"),
		})
		if err != nil {
			return fmt.Errorf("failed to create blob for %s: %w", path, err)
		}

		entries = append(entries, &github.TreeEntry{
			Path: github.String(path),
			Mode: github.String("100644"),
			Type: github.String("blob"),
			SHA:  blob.SHA,
		})
	}

	// Create tree
	tree, _, err := p.client.Git.CreateTree(ctx, p.config.Owner, p.config.Repo, "", entries)
	if err != nil {
		return fmt.Errorf("failed to create tree: %w", err)
	}

	// Create initial commit
	commit, _, err := p.client.Git.CreateCommit(ctx, p.config.Owner, p.config.Repo, &github.Commit{
		Message: github.String(commitMessage),
		Tree:    tree,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	// Create branch reference
	_, _, err = p.client.Git.CreateRef(ctx, p.config.Owner, p.config.Repo, &github.Reference{
		Ref: github.String("refs/heads/" + branch),
		Object: &github.GitObject{
			SHA: commit.SHA,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	fmt.Printf("✅ Created initial commit with %d files\n", len(files))
	return nil
}

func (p *PipelineSetup) CreateDestroyWorkflowFile() error {
	return p.CreateDestroyWorkflowFileOnBranch(p.config.Branch)
}

func (p *PipelineSetup) CreateDestroyWorkflowFileOnBranch(targetBranch string) error {
	workflowPath := ".github/workflows/destroy.yml"
	content := []byte(terraformDestroyTemplate)

	if targetBranch == "" {
		targetBranch = "main"
	}

	fmt.Printf("📝 Creating destroy workflow on branch: %s\n", targetBranch)

	existingFile, _, _, err := p.client.Repositories.GetContents(
		context.Background(),
		p.config.Owner,
		p.config.Repo,
		workflowPath,
		&github.RepositoryContentGetOptions{
			Ref: targetBranch,
		},
	)

	fileOptions := &github.RepositoryContentFileOptions{
		Message: github.String("Add/Update Terraform Destroy workflow"),
		Content: content,
		Branch:  github.String(targetBranch),
	}

	if err == nil && existingFile != nil {
		fmt.Println("📝 Updating existing destroy workflow file")
		fileOptions.SHA = existingFile.SHA
		_, _, err = p.client.Repositories.UpdateFile(
			context.Background(),
			p.config.Owner,
			p.config.Repo,
			workflowPath,
			fileOptions,
		)
	} else {
		fmt.Println("📝 Creating new destroy workflow file")
		_, _, err = p.client.Repositories.CreateFile(
			context.Background(),
			p.config.Owner,
			p.config.Repo,
			workflowPath,
			fileOptions,
		)
	}

	if err != nil {
		fmt.Printf("❌ Failed to create/update destroy workflow: %v\n", err)
		return fmt.Errorf("failed to create destroy workflow: %w", err)
	}

	fmt.Println("✅ Destroy workflow file created/updated successfully")
	return nil
}

func encryptSecret(plaintext, publicKeyB64 string) (string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return "", err
	}

	var publicKey [32]byte
	copy(publicKey[:], publicKeyBytes)

	encrypted, err := box.SealAnonymous(nil, []byte(plaintext), &publicKey, rand.Reader)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func (p *PipelineSetup) setupSecrets() error {
	secrets := map[string]string{
		"AWS_REGION":        p.config.AWSRegion,
		"PIPELINE_ROLE_ARN": p.config.PipelineRoleARN,
	}

	if p.config.TFStateBucket != "" {
		secrets["TF_STATE_BUCKET"] = p.config.TFStateBucket
	}

	if p.config.HasExistingBackend {
		secrets["BACKEND_EXISTS"] = "true"
	} else {
		secrets["BACKEND_EXISTS"] = "false"
	}



	for name, value := range secrets {
		if value == "" {
			continue
		}
		
		pubKey, _, err := p.client.Actions.GetRepoPublicKey(context.Background(), p.config.Owner, p.config.Repo)
		if err != nil {
			return err
		}

		encryptedValue, err := encryptSecret(value, *pubKey.Key)
		if err != nil {
			return err
		}

		_, err = p.client.Actions.CreateOrUpdateRepoSecret(context.Background(), p.config.Owner, p.config.Repo, &github.EncryptedSecret{
			Name:           name,
			EncryptedValue: encryptedValue,
			KeyID:          *pubKey.KeyID,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PipelineSetup) setupEnvironments() error {
	return nil
}

func (p *PipelineSetup) ensureRepository() error {
	_, _, err := p.client.Repositories.Get(context.Background(), p.config.Owner, p.config.Repo)
	if err == nil {
		return nil
	}

	repo := &github.Repository{
		Name:    github.String(p.config.Repo),
		Private: github.Bool(true),
	}

	_, _, err = p.client.Repositories.Create(context.Background(), "", repo)
	return err
}

func (p *PipelineSetup) getExistingBucket() (string, error) {
	// Try to find existing backend.tf to extract bucket name
	backendFile, _, _, err := p.client.Repositories.GetContents(
		context.Background(),
		p.config.Owner,
		p.config.Repo,
		"backend.tf",
		&github.RepositoryContentGetOptions{},
	)
	if err != nil {
		return "", err
	}

	content, err := backendFile.GetContent()
	if err != nil {
		return "", err
	}

	// Extract bucket name from backend.tf
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "bucket") && strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				bucketName := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				return bucketName, nil
			}
		}
	}
	return "", fmt.Errorf("bucket name not found in backend.tf")
}

func (p *PipelineSetup) GetBackendConfig() (bucket, region string, err error) {
	// Try to find existing backend.tf to extract bucket name and region
	backendFile, _, _, err := p.client.Repositories.GetContents(
		context.Background(),
		p.config.Owner,
		p.config.Repo,
		"backend.tf",
		&github.RepositoryContentGetOptions{},
	)
	if err != nil {
		return "", "", err
	}

	content, err := backendFile.GetContent()
	if err != nil {
		return "", "", err
	}

	// Extract bucket name and region from backend.tf
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "bucket") && strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				bucket = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			}
		}
		if strings.Contains(line, "region") && strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				region = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			}
		}
	}

	if bucket == "" {
		return "", "", fmt.Errorf("bucket name not found in backend.tf")
	}
	if region == "" {
		return "", "", fmt.Errorf("region not found in backend.tf")
	}

	return bucket, region, nil
}