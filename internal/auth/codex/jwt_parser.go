package codex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// JWTClaims represents the claims section of a JWT
type JWTClaims struct {
	AtHash        string        `json:"at_hash"`
	Aud           []string      `json:"aud"`
	AuthProvider  string        `json:"auth_provider"`
	AuthTime      int           `json:"auth_time"`
	Email         string        `json:"email"`
	EmailVerified bool          `json:"email_verified"`
	Exp           int           `json:"exp"`
	CodexAuthInfo CodexAuthInfo `json:"https://api.openai.com/auth"`
	Iat           int           `json:"iat"`
	Iss           string        `json:"iss"`
	Jti           string        `json:"jti"`
	Rat           int           `json:"rat"`
	Sid           string        `json:"sid"`
	Sub           string        `json:"sub"`
}

// Organizations defines organization details within JWT claims
type Organizations struct {
	ID        string `json:"id"`
	IsDefault bool   `json:"is_default"`
	Role      string `json:"role"`
	Title     string `json:"title"`
}

// CodexAuthInfo contains Codex-specific authentication details
type CodexAuthInfo struct {
	ChatGptAccountID               string          `json:"chatgpt_account_id"`
	ChatGptPlanType                string          `json:"chatgpt_plan_type"`
	ChatGptSubscriptionActiveStart interface{}     `json:"chatgpt_subscription_active_start"`
	ChatGptSubscriptionActiveUntil interface{}     `json:"chatgpt_subscription_active_until"`
	ChatGptSubscriptionLastChecked string          `json:"chatgpt_subscription_last_checked"`
	ChatGptUserID                  string          `json:"chatgpt_user_id"`
	Groups                         []interface{}   `json:"groups"`
	Organizations                  []Organizations `json:"organizations"`
	UserID                         string          `json:"user_id"`
}

// ParseJWTToken parses a JWT token string and extracts its claims
func ParseJWTToken(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format: expected 3 parts, got %d", len(parts))
	}

	claimsData, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT claims: %w", err)
	}

	var claims JWTClaims
	if err = json.Unmarshal(claimsData, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT claims: %w", err)
	}

	return &claims, nil
}

// base64URLDecode decodes a Base64 URL-encoded string
func base64URLDecode(data string) ([]byte, error) {
	switch len(data) % 4 {
	case 2:
		data += "=="
	case 3:
		data += "="
	}
	return base64.URLEncoding.DecodeString(data)
}

// GetUserEmail extracts the user's email from JWT claims
func (c *JWTClaims) GetUserEmail() string {
	return c.Email
}

// GetAccountID extracts the ChatGPT account ID from JWT claims
func (c *JWTClaims) GetAccountID() string {
	return c.CodexAuthInfo.ChatGptAccountID
}
