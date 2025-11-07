package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubAuth struct {
	config *oauth2.Config
	state  string
	token  *oauth2.Token
}

func NewGitHubAuth(clientID, clientSecret string) *GitHubAuth {
	state := generateState()
	
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"repo", "workflow", "admin:repo_hook", "read:org"},
		Endpoint:     github.Endpoint,
		RedirectURL:  "http://localhost:8080/callback",
	}

	return &GitHubAuth{
		config: config,
		state:  state,
	}
}

func (g *GitHubAuth) GetAuthURL() string {
	return g.config.AuthCodeURL(g.state, oauth2.AccessTypeOffline)
}

func (g *GitHubAuth) HandleCallback(code, state string) error {
	if state != g.state {
		return fmt.Errorf("invalid state parameter")
	}

	token, err := g.config.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	g.token = token
	return nil
}

func (g *GitHubAuth) GetToken() string {
	if g.token == nil {
		return ""
	}
	return g.token.AccessToken
}

func (g *GitHubAuth) StartServer() error {
	done := make(chan error, 1)
	
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		
		if err := g.HandleCallback(code, state); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			done <- err
			return
		}
		
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<body>
					<h2>✅ Authentication Successful!</h2>
					<p>You can close this window and return to the terminal.</p>
					<script>setTimeout(() => window.close(), 2000);</script>
				</body>
			</html>
		`))
		done <- nil
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			done <- err
		}
	}()

	select {
	case err := <-done:
		server.Shutdown(context.Background())
		return err
	case <-time.After(5 * time.Minute):
		server.Shutdown(context.Background())
		return fmt.Errorf("authentication timeout")
	}
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}