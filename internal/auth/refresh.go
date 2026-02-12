package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/auth/antigravity"
	"github.com/bestruirui/octopus/internal/auth/codex"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/utils/log"
)

// TokenRefresher handles automatic token refresh
type TokenRefresher struct {
	codexAuth      *codex.CodexAuth
	antigravityAuth *antigravity.AntigravityAuth
	refreshLock    sync.Map // Prevent concurrent refresh for same token
}

// NewTokenRefresher creates a new token refresher
func NewTokenRefresher() *TokenRefresher {
	return &TokenRefresher{
		codexAuth:       codex.NewCodexAuth(),
		antigravityAuth: antigravity.NewAntigravityAuth(),
	}
}

// RefreshTokenIfNeeded checks if token needs refresh and refreshes it
func (r *TokenRefresher) RefreshTokenIfNeeded(token *model.OAuthToken, ctx context.Context) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	if !token.NeedsRefresh() {
		return nil
	}

	// Prevent concurrent refresh for the same token
	if _, loaded := r.refreshLock.LoadOrStore(token.ID, true); loaded {
		// Another goroutine is already refreshing this token
		return nil
	}
	defer r.refreshLock.Delete(token.ID)

	log.Infof("Refreshing OAuth token %d (type: %s, email: %s)", token.ID, token.Type, token.Email)

	switch token.Type {
	case model.OAuthTypeCodex:
		return r.refreshCodexToken(token, ctx)
	case model.OAuthTypeAntigravity:
		return r.refreshAntigravityToken(token, ctx)
	default:
		return fmt.Errorf("unknown token type: %s", token.Type)
	}
}

// refreshCodexToken refreshes a Codex OAuth token
func (r *TokenRefresher) refreshCodexToken(token *model.OAuthToken, ctx context.Context) error {
	tokenResp, _, err := r.codexAuth.RefreshTokens(ctx, token.RefreshToken)
	if err != nil {
		log.Errorf("Failed to refresh Codex token %d: %v", token.ID, err)
		return fmt.Errorf("failed to refresh codex token: %w", err)
	}

	// Update token in database
	if err := op.OAuthTokenRefresh(token.ID, tokenResp.AccessToken, tokenResp.RefreshToken, int64(tokenResp.ExpiresIn), ctx); err != nil {
		log.Errorf("Failed to update Codex token %d in database: %v", token.ID, err)
		return fmt.Errorf("failed to update token in database: %w", err)
	}

	log.Infof("Successfully refreshed Codex token %d", token.ID)
	return nil
}

// refreshAntigravityToken refreshes an Antigravity OAuth token
func (r *TokenRefresher) refreshAntigravityToken(token *model.OAuthToken, ctx context.Context) error {
	tokenResp, err := r.antigravityAuth.RefreshTokens(ctx, token.RefreshToken)
	if err != nil {
		log.Errorf("Failed to refresh Antigravity token %d: %v", token.ID, err)
		return fmt.Errorf("failed to refresh antigravity token: %w", err)
	}

	// Update token in database
	if err := op.OAuthTokenRefresh(token.ID, tokenResp.AccessToken, tokenResp.RefreshToken, tokenResp.ExpiresIn, ctx); err != nil {
		log.Errorf("Failed to update Antigravity token %d in database: %v", token.ID, err)
		return fmt.Errorf("failed to update token in database: %w", err)
	}

	log.Infof("Successfully refreshed Antigravity token %d", token.ID)
	return nil
}

// Global token refresher instance
var globalRefresher = NewTokenRefresher()

// RefreshTokenIfNeeded is a convenience function using the global refresher
func RefreshTokenIfNeeded(token *model.OAuthToken, ctx context.Context) error {
	return globalRefresher.RefreshTokenIfNeeded(token, ctx)
}

// StartBackgroundRefresh starts a background goroutine to periodically check and refresh tokens
func StartBackgroundRefresh(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		interval = 5 * time.Minute
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refreshAllTokens(ctx)
			}
		}
	}()
}

// refreshAllTokens checks and refreshes all tokens that need refresh
func refreshAllTokens(ctx context.Context) {
	tokens, err := op.OAuthTokenList(ctx)
	if err != nil {
		log.Warnf("Failed to get token list for background refresh: %v", err)
		return
	}

	for _, tokenResp := range tokens {
		if !tokenResp.Enabled {
			continue
		}

		// Get full token from cache
		token, err := op.OAuthTokenGet(tokenResp.ID, ctx)
		if err != nil {
			continue
		}

		if token.NeedsRefresh() {
			if err := globalRefresher.RefreshTokenIfNeeded(token, ctx); err != nil {
				log.Warnf("Background refresh failed for token %d: %v", token.ID, err)
			}
		}
	}
}
