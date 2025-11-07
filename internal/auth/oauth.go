package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GitHubOAuth struct {
	clientID     string
	clientSecret string
	redirectURL  string
	state        string
	token        string
	server       *http.Server
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func NewGitHubOAuth() *GitHubOAuth {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	return &GitHubOAuth{
		clientID:     "Ov23lixAgV5sEdMtSTDL", // Your OAuth App
		clientSecret: "af3eda8371432e81483f2d8a6cd7cd73319d2c88", // Your OAuth secret
		redirectURL:  "http://localhost:8080/callback",
		state:        state,
	}
}

func (g *GitHubOAuth) GetAuthURL() string {
	params := url.Values{}
	params.Add("client_id", g.clientID)
	params.Add("redirect_uri", g.redirectURL)
	params.Add("scope", "repo workflow admin:repo_hook")
	params.Add("state", g.state)
	
	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

func (g *GitHubOAuth) StartServer() error {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Callback received: %s\n", r.URL.String())
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		errorParam := r.URL.Query().Get("error")
		
		if errorParam != "" {
			fmt.Printf("OAuth error: %s\n", errorParam)
			http.Error(w, "OAuth error: "+errorParam, http.StatusBadRequest)
			return
		}
		
		if state != g.state {
			fmt.Printf("State mismatch: expected %s, got %s\n", g.state, state)
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}
		
		if code == "" {
			fmt.Println("No authorization code received")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}
		
		fmt.Printf("Exchanging code for token...\n")
		token, err := g.exchangeCodeForToken(code)
		if err != nil {
			fmt.Printf("Token exchange failed: %v\n", err)
			http.Error(w, "Failed to get token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		g.token = token
		fmt.Printf("Token received successfully\n")
		
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
			<head><title>Authentication Successful</title></head>
			<body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
				<h1 style="color: green;">✅ Authentication Successful!</h1>
				<p>You have successfully signed in to GitHub.</p>
				<p>You can now close this window and return to the terminal.</p>
				<script>
					setTimeout(function() {
						window.close();
					}, 3000);
				</script>
			</body>
			</html>
		`)
		
		go func() {
			time.Sleep(2 * time.Second)
			g.server.Shutdown(context.Background())
		}()
	})
	
	g.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	
	return g.server.ListenAndServe()
}

func (g *GitHubOAuth) exchangeCodeForToken(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", g.clientID)
	data.Set("client_secret", g.clientSecret)
	data.Set("code", code)
	
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	fmt.Printf("Token response: %s\n", string(body))
	
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w - body: %s", err, string(body))
	}
	
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response: %s", string(body))
	}
	
	return tokenResp.AccessToken, nil
}

func (g *GitHubOAuth) GetToken() string {
	return g.token
}