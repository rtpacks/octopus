package antigravity

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

// TokenResponse represents OAuth token response from Google
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// userInfo represents Google user profile
type userInfo struct {
	Email string `json:"email"`
}

// AntigravityAuth handles Antigravity OAuth authentication
type AntigravityAuth struct {
	httpClient *http.Client
}

// NewAntigravityAuth creates a new Antigravity auth service
func NewAntigravityAuth() *AntigravityAuth {
	return &AntigravityAuth{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// BuildAuthURL generates the OAuth authorization URL
func (o *AntigravityAuth) BuildAuthURL(state, redirectURI string) string {
	if strings.TrimSpace(redirectURI) == "" {
		redirectURI = fmt.Sprintf("http://localhost:%d/oauth-callback", CallbackPort)
	}

	params := url.Values{}
	params.Set("access_type", "offline")
	params.Set("client_id", ClientID)
	params.Set("prompt", "consent")
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(Scopes, " "))
	params.Set("state", state)

	return AuthEndpoint + "?" + params.Encode()
}

// ExchangeCodeForTokens exchanges authorization code for tokens
func (o *AntigravityAuth) ExchangeCodeForTokens(ctx context.Context, code, redirectURI string) (*TokenResponse, error) {
	if strings.TrimSpace(redirectURI) == "" {
		redirectURI = fmt.Sprintf("http://localhost:%d/oauth-callback", CallbackPort)
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", ClientID)
	data.Set("client_secret", ClientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("antigravity token exchange: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("antigravity token exchange: execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		body := strings.TrimSpace(string(bodyBytes))
		if body == "" {
			return nil, fmt.Errorf("antigravity token exchange: request failed: status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("antigravity token exchange: request failed: status %d: %s", resp.StatusCode, body)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("antigravity token exchange: decode response: %w", err)
	}

	return &token, nil
}

// RefreshTokens refreshes an access token using a refresh token
func (o *AntigravityAuth) RefreshTokens(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", ClientID)
	data.Set("client_secret", ClientSecret)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("antigravity token refresh: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("antigravity token refresh: execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("antigravity token refresh: status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("antigravity token refresh: decode response: %w", err)
	}

	// Preserve refresh token if not returned
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}

	return &token, nil
}

// FetchUserInfo retrieves user email from Google
func (o *AntigravityAuth) FetchUserInfo(ctx context.Context, accessToken string) (string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return "", fmt.Errorf("antigravity userinfo: missing access token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, UserInfoEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("antigravity userinfo: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("antigravity userinfo: execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return "", fmt.Errorf("antigravity userinfo: status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("antigravity userinfo: decode response: %w", err)
	}

	email := strings.TrimSpace(info.Email)
	if email == "" {
		return "", fmt.Errorf("antigravity userinfo: response missing email")
	}

	return email, nil
}

// FetchProjectID retrieves the project ID for the authenticated user
func (o *AntigravityAuth) FetchProjectID(ctx context.Context, accessToken string) (string, error) {
	loadReqBody := map[string]any{
		"metadata": map[string]string{
			"ideType":    "ANTIGRAVITY",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}

	rawBody, err := json.Marshal(loadReqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	endpointURL := fmt.Sprintf("%s/%s:loadCodeAssist", APIEndpoint, APIVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, strings.NewReader(string(rawBody)))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", APIUserAgent)
	req.Header.Set("X-Goog-Api-Client", APIClient)
	req.Header.Set("Client-Metadata", ClientMetadata)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var loadResp map[string]any
	if err := json.Unmarshal(bodyBytes, &loadResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	// Extract projectID from response
	projectID := extractProjectID(loadResp)
	if projectID == "" {
		// Try to onboard user
		tierID := "legacy-tier"
		if tiers, ok := loadResp["allowedTiers"].([]any); ok {
			for _, rawTier := range tiers {
				if tier, ok := rawTier.(map[string]any); ok {
					if isDefault, ok := tier["isDefault"].(bool); ok && isDefault {
						if id, ok := tier["id"].(string); ok && strings.TrimSpace(id) != "" {
							tierID = strings.TrimSpace(id)
							break
						}
					}
				}
			}
		}

		return o.OnboardUser(ctx, accessToken, tierID)
	}

	return projectID, nil
}

// extractProjectID extracts project ID from loadCodeAssist response
func extractProjectID(loadResp map[string]any) string {
	if id, ok := loadResp["cloudaicompanionProject"].(string); ok {
		return strings.TrimSpace(id)
	}
	if projectMap, ok := loadResp["cloudaicompanionProject"].(map[string]any); ok {
		if id, okID := projectMap["id"].(string); okID {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

// OnboardUser attempts to onboard user and get project ID
func (o *AntigravityAuth) OnboardUser(ctx context.Context, accessToken, tierID string) (string, error) {
	log.Infof("Antigravity: onboarding user with tier: %s", tierID)

	requestBody := map[string]any{
		"tierId": tierID,
		"metadata": map[string]string{
			"ideType":    "ANTIGRAVITY",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}

	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Debugf("Polling attempt %d/%d", attempt, maxAttempts)

		reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		endpointURL := fmt.Sprintf("%s/%s:onboardUser", APIEndpoint, APIVersion)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpointURL, strings.NewReader(string(rawBody)))
		if err != nil {
			cancel()
			return "", fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", APIUserAgent)
		req.Header.Set("X-Goog-Api-Client", APIClient)
		req.Header.Set("Client-Metadata", ClientMetadata)

		resp, err := o.httpClient.Do(req)
		if err != nil {
			cancel()
			return "", fmt.Errorf("execute request: %w", err)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()

		if err != nil {
			return "", fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			var data map[string]any
			if err := json.Unmarshal(bodyBytes, &data); err != nil {
				return "", fmt.Errorf("decode response: %w", err)
			}

			if done, ok := data["done"].(bool); ok && done {
				projectID := ""
				if responseData, ok := data["response"].(map[string]any); ok {
					switch projectValue := responseData["cloudaicompanionProject"].(type) {
					case map[string]any:
						if id, okID := projectValue["id"].(string); okID {
							projectID = strings.TrimSpace(id)
						}
					case string:
						projectID = strings.TrimSpace(projectValue)
					}
				}

				if projectID != "" {
					log.Infof("Successfully fetched project_id: %s", projectID)
					return projectID, nil
				}

				return "", fmt.Errorf("no project_id in response")
			}

			time.Sleep(2 * time.Second)
			continue
		}

		responseErr := strings.TrimSpace(string(bodyBytes))
		if len(responseErr) > 200 {
			responseErr = responseErr[:200]
		}
		return "", fmt.Errorf("http %d: %s", resp.StatusCode, responseErr)
	}

	return "", fmt.Errorf("onboarding failed after %d attempts", maxAttempts)
}

// CreateOAuthToken creates an OAuthToken model from token response
func CreateOAuthToken(tokenResp *TokenResponse, email string) *model.OAuthToken {
	token := &model.OAuthToken{
		Type:         model.OAuthTypeAntigravity,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		Email:        email,
		Enabled:      true,
	}

	if tokenResp.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		token.ExpiresAt = &expiresAt
	}

	now := time.Now()
	token.LastRefresh = &now

	return token
}
