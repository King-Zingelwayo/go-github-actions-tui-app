package github

import (
	"context"
	"encoding/base64"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *github.Client
	ctx    context.Context
}

func NewClient(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	
	return &Client{
		client: github.NewClient(tc),
		ctx:    ctx,
	}
}

func (c *Client) CreateRepository(owner, name, description string, private bool) error {
	repo := &github.Repository{
		Name:        github.String(name),
		Description: github.String(description),
		Private:     github.Bool(private),
	}

	_, _, err := c.client.Repositories.Create(c.ctx, "", repo)
	return err
}

func (c *Client) CreateSecret(owner, repo, name, value string) error {
	// Get repository public key
	key, _, err := c.client.Actions.GetRepoPublicKey(c.ctx, owner, repo)
	if err != nil {
		return err
	}

	// Encrypt the secret value
	encryptedValue, err := encryptSecret(value, key.GetKey())
	if err != nil {
		return err
	}

	// Create the secret
	secret := &github.EncryptedSecret{
		Name:           name,
		KeyID:          key.GetKeyID(),
		EncryptedValue: encryptedValue,
	}

	_, err = c.client.Actions.CreateOrUpdateRepoSecret(c.ctx, owner, repo, secret)
	return err
}

func (c *Client) CreateWorkflowFile(owner, repo, path, content string) error {
	opts := &github.RepositoryContentFileOptions{
		Message: github.String("Add Terraform CI/CD workflow"),
		Content: []byte(content),
	}

	_, _, err := c.client.Repositories.CreateFile(c.ctx, owner, repo, path, opts)
	return err
}

func encryptSecret(value, publicKey string) (string, error) {
	// This is a simplified version - in production, use proper NaCl encryption
	// For now, just base64 encode (NOT SECURE - just for demo)
	return base64.StdEncoding.EncodeToString([]byte(value)), nil
}