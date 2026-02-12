package op

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
)

var oauthTokenCache = cache.New[uint, model.OAuthToken](4)

// OAuthTokenList returns all OAuth tokens (without sensitive data)
func OAuthTokenList(ctx context.Context) ([]model.OAuthTokenResponse, error) {
	tokens := oauthTokenCache.GetAll()
	result := make([]model.OAuthTokenResponse, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, token.ToResponse())
	}
	return result, nil
}

// OAuthTokenGet returns an OAuth token by ID
func OAuthTokenGet(id uint, ctx context.Context) (*model.OAuthToken, error) {
	token, ok := oauthTokenCache.Get(id)
	if !ok {
		return nil, fmt.Errorf("oauth token not found")
	}
	return &token, nil
}

// OAuthTokenCreate creates a new OAuth token
func OAuthTokenCreate(req *model.OAuthTokenCreateRequest, ctx context.Context) (*model.OAuthToken, error) {
	token := &model.OAuthToken{
		Type:         req.Type,
		Email:        req.Email,
		AccessToken:  req.AccessToken,
		RefreshToken: req.RefreshToken,
		IDToken:      req.IDToken,
		Enabled:      true,
		Remark:       req.Remark,
	}

	if req.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		token.ExpiresAt = &expiresAt
	}

	now := time.Now()
	token.LastRefresh = &now

	if err := db.GetDB().WithContext(ctx).Create(token).Error; err != nil {
		return nil, fmt.Errorf("failed to create oauth token: %w", err)
	}

	oauthTokenCache.Set(token.ID, *token)
	return token, nil
}

// OAuthTokenUpdate updates an OAuth token
func OAuthTokenUpdate(req *model.OAuthTokenUpdateRequest, ctx context.Context) (*model.OAuthToken, error) {
	token, ok := oauthTokenCache.Get(req.ID)
	if !ok {
		return nil, fmt.Errorf("oauth token not found")
	}

	updates := map[string]interface{}{}

	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
		token.Enabled = *req.Enabled
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
		token.Remark = req.Remark
	}
	if req.AccessToken != "" {
		updates["access_token"] = req.AccessToken
		token.AccessToken = req.AccessToken
	}
	if req.RefreshToken != "" {
		updates["refresh_token"] = req.RefreshToken
		token.RefreshToken = req.RefreshToken
	}
	if req.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		updates["expires_at"] = expiresAt
		token.ExpiresAt = &expiresAt

		now := time.Now()
		updates["last_refresh"] = now
		token.LastRefresh = &now
	}

	if len(updates) > 0 {
		if err := db.GetDB().WithContext(ctx).Model(&model.OAuthToken{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update oauth token: %w", err)
		}
		oauthTokenCache.Set(token.ID, token)
	}

	return &token, nil
}

// OAuthTokenDelete deletes an OAuth token
func OAuthTokenDelete(id uint, ctx context.Context) error {
	_, ok := oauthTokenCache.Get(id)
	if !ok {
		return fmt.Errorf("oauth token not found")
	}

	// Check if any channel is using this token
	var count int64
	if err := db.GetDB().WithContext(ctx).Model(&model.Channel{}).Where("oauth_token_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check channel usage: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete oauth token: %d channels are using it", count)
	}

	if err := db.GetDB().WithContext(ctx).Delete(&model.OAuthToken{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete oauth token: %w", err)
	}

	oauthTokenCache.Del(id)
	return nil
}

// OAuthTokenRefresh updates the token after a successful refresh
func OAuthTokenRefresh(id uint, accessToken, refreshToken string, expiresIn int64, ctx context.Context) error {
	token, ok := oauthTokenCache.Get(id)
	if !ok {
		return fmt.Errorf("oauth token not found")
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(expiresIn) * time.Second)

	updates := map[string]interface{}{
		"access_token": accessToken,
		"expires_at":   expiresAt,
		"last_refresh": now,
	}

	if refreshToken != "" {
		updates["refresh_token"] = refreshToken
	}

	if err := db.GetDB().WithContext(ctx).Model(&model.OAuthToken{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to refresh oauth token: %w", err)
	}

	token.AccessToken = accessToken
	token.ExpiresAt = &expiresAt
	token.LastRefresh = &now
	if refreshToken != "" {
		token.RefreshToken = refreshToken
	}

	oauthTokenCache.Set(id, token)
	return nil
}

// oauthTokenRefreshCache refreshes the OAuth token cache from database
func oauthTokenRefreshCache(ctx context.Context) error {
	tokens := []model.OAuthToken{}
	if err := db.GetDB().WithContext(ctx).Find(&tokens).Error; err != nil {
		return fmt.Errorf("failed to get oauth tokens: %w", err)
	}

	oauthTokenCache.Clear()
	for _, token := range tokens {
		oauthTokenCache.Set(token.ID, token)
	}
	return nil
}

// InitOAuthTokenCache initializes the OAuth token cache
func InitOAuthTokenCache(ctx context.Context) error {
	return oauthTokenRefreshCache(ctx)
}

var oauthTokenRefreshLock sync.Mutex

// RefreshExpiredOAuthTokens checks and refreshes expired OAuth tokens
func RefreshExpiredOAuthTokens(ctx context.Context) error {
	oauthTokenRefreshLock.Lock()
	defer oauthTokenRefreshLock.Unlock()

	tokens := oauthTokenCache.GetAll()
	for _, token := range tokens {
		if !token.Enabled {
			continue
		}
		if token.NeedsRefresh() {
			// Token needs refresh - this will be handled by the auth module
			// For now, just log a warning
			// The actual refresh logic will be implemented in Phase 2
		}
	}
	return nil
}
