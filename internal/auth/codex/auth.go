package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/log"
)

// CodexAuth handles OpenAI OAuth2 authentication
type CodexAuth struct {
	httpClient *http.Client
}

// NewCodexAuth creates a new CodexAuth instance
func NewCodexAuth() *CodexAuth {
	return &CodexAuth{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// TokenResponse represents the token response from OpenAI
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// GenerateAuthURL creates the OAuth authorization URL with PKCE
func (o *CodexAuth) GenerateAuthURL(state string, pkceCodes *PKCECodes) (string, error) {
	if pkceCodes == nil {
		return "", fmt.Errorf("PKCE codes are required")
	}

	params := url.Values{
		"client_id":                  {ClientID},
		"response_type":              {"code"},
		"redirect_uri":               {RedirectURI},
		"scope":                      {"openid email profile offline_access"},
		"state":                      {state},
		"code_challenge":             {pkceCodes.CodeChallenge},
		"code_challenge_method":      {"S256"},
		"prompt":                     {"login"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
	}

	return fmt.Sprintf("%s?%s", AuthURL, params.Encode()), nil
}

// ExchangeCodeForTokens exchanges an authorization code for tokens
func (o *CodexAuth) ExchangeCodeForTokens(ctx context.Context, code string, pkceCodes *PKCECodes) (*TokenResponse, *JWTClaims, error) {
	if pkceCodes == nil {
		return nil, nil, fmt.Errorf("PKCE codes are required for token exchange")
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {ClientID},
		"code":          {code},
		"redirect_uri":  {RedirectURI},
		"code_verifier": {pkceCodes.CodeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Parse ID token to get user info
	claims, err := ParseJWTToken(tokenResp.IDToken)
	if err != nil {
		log.Warnf("Failed to parse ID token: %v", err)
	}

	return &tokenResp, claims, nil
}

// RefreshTokens refreshes an access token using a refresh token
func (o *CodexAuth) RefreshTokens(ctx context.Context, refreshToken string) (*TokenResponse, *JWTClaims, error) {
	if refreshToken == "" {
		return nil, nil, fmt.Errorf("refresh token is required")
	}

	data := url.Values{
		"client_id":     {ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {"openid profile email"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	claims, err := ParseJWTToken(tokenResp.IDToken)
	if err != nil {
		log.Warnf("Failed to parse refreshed ID token: %v", err)
	}

	return &tokenResp, claims, nil
}

// CreateOAuthToken creates an OAuthToken model from token response
func CreateOAuthToken(tokenResp *TokenResponse, claims *JWTClaims) *model.OAuthToken {
	token := &model.OAuthToken{
		Type:         model.OAuthTypeCodex,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		Enabled:      true,
	}

	if claims != nil {
		token.Email = claims.Email
	}

	if tokenResp.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		token.ExpiresAt = &expiresAt
	}

	now := time.Now()
	token.LastRefresh = &now

	return token
}

// RefreshTokensWithRetry refreshes tokens with retry mechanism
func (o *CodexAuth) RefreshTokensWithRetry(ctx context.Context, refreshToken string, maxRetries int) (*TokenResponse, *JWTClaims, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		tokenResp, claims, err := o.RefreshTokens(ctx, refreshToken)
		if err == nil {
			return tokenResp, claims, nil
		}

		lastErr = err
		log.Warnf("Token refresh attempt %d failed: %v", attempt+1, err)
	}

	return nil, nil, fmt.Errorf("token refresh failed after %d attempts: %w", maxRetries, lastErr)
}
