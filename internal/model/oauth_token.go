package model

import (
	"time"
)

// OAuthTokenType defines the OAuth provider type
type OAuthTokenType string

const (
	OAuthTypeCodex      OAuthTokenType = "codex"
	OAuthTypeAntigravity OAuthTokenType = "antigravity"
)

// OAuthToken stores OAuth authentication tokens for channels
type OAuthToken struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Type         OAuthTokenType `json:"type" gorm:"index;not null"`
	Email        string         `json:"email" gorm:"index"`
	AccessToken  string         `json:"-" gorm:"type:text"`  // Hidden from JSON response
	RefreshToken string         `json:"-" gorm:"type:text"`  // Hidden from JSON response
	IDToken      string         `json:"-" gorm:"type:text"`  // Hidden from JSON response
	ExpiresAt    *time.Time     `json:"expires_at"`
	LastRefresh  *time.Time     `json:"last_refresh"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Enabled      bool           `json:"enabled" gorm:"default:true"`
	Remark       string         `json:"remark"`
}

// TableName specifies the table name for OAuthToken
func (OAuthToken) TableName() string {
	return "oauth_tokens"
}

// IsExpired checks if the token is expired
func (t *OAuthToken) IsExpired() bool {
	if t == nil || t.ExpiresAt == nil {
		return true
	}
	return time.Now().After(*t.ExpiresAt)
}

// NeedsRefresh checks if the token needs to be refreshed (5 minutes before expiry)
func (t *OAuthToken) NeedsRefresh() bool {
	if t == nil || t.ExpiresAt == nil {
		return true
	}
	return time.Now().Add(5 * time.Minute).After(*t.ExpiresAt)
}

// OAuthTokenCreateRequest for creating new OAuth tokens
type OAuthTokenCreateRequest struct {
	Type         OAuthTokenType `json:"type" binding:"required"`
	Email        string         `json:"email"`
	AccessToken  string         `json:"access_token" binding:"required"`
	RefreshToken string         `json:"refresh_token"`
	IDToken      string         `json:"id_token"`
	ExpiresIn    int64          `json:"expires_in"` // Expires in seconds
	Remark       string         `json:"remark"`
}

// OAuthTokenUpdateRequest for updating OAuth tokens
type OAuthTokenUpdateRequest struct {
	ID           uint   `json:"id" binding:"required"`
	Enabled      *bool  `json:"enabled,omitempty"`
	Remark       string `json:"remark,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
}

// OAuthTokenResponse for API responses (sensitive fields hidden)
type OAuthTokenResponse struct {
	ID          uint           `json:"id"`
	Type        OAuthTokenType `json:"type"`
	Email       string         `json:"email"`
	ExpiresAt   *time.Time     `json:"expires_at"`
	LastRefresh *time.Time     `json:"last_refresh"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Enabled     bool           `json:"enabled"`
	Remark      string         `json:"remark"`
	IsExpired   bool           `json:"is_expired"`
}

// ToResponse converts OAuthToken to OAuthTokenResponse
func (t *OAuthToken) ToResponse() OAuthTokenResponse {
	return OAuthTokenResponse{
		ID:          t.ID,
		Type:        t.Type,
		Email:       t.Email,
		ExpiresAt:   t.ExpiresAt,
		LastRefresh: t.LastRefresh,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		Enabled:     t.Enabled,
		Remark:      t.Remark,
		IsExpired:   t.IsExpired(),
	}
}
