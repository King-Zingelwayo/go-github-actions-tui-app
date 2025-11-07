package auth

import (
	"fmt"
	"io"
	"net/http"
)

type GitHubBasicAuth struct {
	username string
	password string
}

func NewGitHubBasicAuth(username, password string) *GitHubBasicAuth {
	return &GitHubBasicAuth{
		username: username,
		password: password,
	}
}

func (g *GitHubBasicAuth) CreateToken() (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "token "+g.password)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return "", fmt.Errorf("invalid token - please create a personal access token at https://github.com/settings/tokens")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	return g.password, nil
}