package github

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// AppAuth handles GitHub App authentication
type AppAuth struct {
	AppID          int64
	InstallationID int64
	PrivateKey     *rsa.PrivateKey
	BaseURL        string
}

// NewAppAuth creates a new GitHub App authenticator
func NewAppAuth(appID int64, installationID int64, privateKeyPEM []byte, baseURL string) (*AppAuth, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &AppAuth{
		AppID:          appID,
		InstallationID: installationID,
		PrivateKey:     privateKey,
		BaseURL:        baseURL,
	}, nil
}

// GenerateJWT generates a JWT token for GitHub App authentication
func (a *AppAuth) GenerateJWT() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Add(-60 * time.Second).Unix(), // Issued at time (60 seconds in the past to account for clock skew)
		"exp": now.Add(10 * time.Minute).Unix(),  // Expires in 10 minutes
		"iss": a.AppID,                           // Issuer (App ID)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(a.PrivateKey)
}

// GetInstallationToken gets an installation access token for the GitHub App
func (a *AppAuth) GetInstallationToken(ctx context.Context) (string, error) {
	jwtToken, err := a.GenerateJWT()
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Determine API base URL
	apiBaseURL := "https://api.github.com"
	if a.BaseURL != "" && a.BaseURL != "https://api.github.com" {
		apiBaseURL = a.BaseURL
	}

	// Create request to get installation token
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", apiBaseURL, a.InstallationID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get installation token: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return tokenResponse.Token, nil
}

// CreateOAuth2TokenSource creates an oauth2.TokenSource that automatically refreshes installation tokens
func (a *AppAuth) CreateOAuth2TokenSource(ctx context.Context) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, &appTokenSource{
		appAuth: a,
		ctx:     ctx,
	})
}

// appTokenSource implements oauth2.TokenSource for GitHub App tokens
type appTokenSource struct {
	appAuth *AppAuth
	ctx     context.Context
	token   *oauth2.Token
	expires time.Time
}

func (ts *appTokenSource) Token() (*oauth2.Token, error) {
	// Check if we have a valid token that hasn't expired
	now := time.Now()
	if ts.token != nil && ts.expires.After(now.Add(5*time.Minute)) {
		// Token is still valid (with 5 minute buffer)
		return ts.token, nil
	}

	// Get a new installation token
	tokenString, err := ts.appAuth.GetInstallationToken(ts.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation token: %w", err)
	}

	// Installation tokens expire in 1 hour
	ts.expires = time.Now().Add(55 * time.Minute) // Use 55 minutes to be safe
	ts.token = &oauth2.Token{
		AccessToken: tokenString,
		TokenType:   "token",
		Expiry:      ts.expires,
	}

	return ts.token, nil
}

// CreateGitHubClient creates a GitHub client using App authentication
func CreateGitHubClientWithApp(ctx context.Context, appAuth *AppAuth) (*github.Client, error) {
	tokenSource := appAuth.CreateOAuth2TokenSource(ctx)
	tc := oauth2.NewClient(ctx, tokenSource)

	var client *github.Client
	if appAuth.BaseURL != "" && appAuth.BaseURL != "https://api.github.com" {
		var err error
		client, err = github.NewClient(tc).WithEnterpriseURLs(appAuth.BaseURL, appAuth.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub Enterprise client: %w", err)
		}
	} else {
		client = github.NewClient(tc)
	}

	return client, nil
}
