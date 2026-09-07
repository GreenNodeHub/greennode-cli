package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const IAMTokenURL = "https://iamapis.vngcloud.vn/accounts-api/v1/auth/token"

// tokenResponse matches the IAM API camelCase JSON response.
type tokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"`
}

// TokenManager handles OAuth2 Client Credentials flow with auto-refresh.
type TokenManager struct {
	clientID     string
	clientSecret string
	accessToken  string
	expiresAt    time.Time
	mu           sync.Mutex
	httpClient   *http.Client
}

// NewTokenManager creates a new token manager.
func NewTokenManager(clientID, clientSecret string) *TokenManager {
	return &TokenManager{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// GetToken returns a valid access token, fetching a new one if needed.
func (tm *TokenManager) GetToken() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.accessToken != "" && time.Now().Before(tm.expiresAt) {
		return tm.accessToken, nil
	}
	return tm.fetchToken()
}

// SetToken pre-seeds the token manager with a static token and expiry.
// Intended for use in tests only.
func (tm *TokenManager) SetToken(token string, expiresAt time.Time) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.accessToken = token
	tm.expiresAt = expiresAt
}

// RefreshToken forces a token refresh.
func (tm *TokenManager) RefreshToken() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.accessToken = ""
	tm.expiresAt = time.Time{}
	return tm.fetchToken()
}

func (tm *TokenManager) fetchToken() (string, error) {
	credentials := base64.StdEncoding.EncodeToString(
		[]byte(tm.clientID + ":" + tm.clientSecret),
	)

	data := url.Values{}
	data.Set("grantType", "client_credentials")

	req, err := http.NewRequest("POST", IAMTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+credentials)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tm.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("IAM authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IAM authentication error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	tm.accessToken = tokenResp.AccessToken
	expiresIn := tokenResp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 1800
	}
	// Refresh 60 seconds before expiry
	tm.expiresAt = time.Now().Add(time.Duration(expiresIn-60) * time.Second)

	return tm.accessToken, nil
}
