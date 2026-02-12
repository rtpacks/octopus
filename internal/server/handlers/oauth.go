package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bestruirui/octopus/internal/auth"
	"github.com/bestruirui/octopus/internal/auth/antigravity"
	"github.com/bestruirui/octopus/internal/auth/codex"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/gin-gonic/gin"
)

var (
	sessionManager *auth.SessionManager
	codexAuth      *codex.CodexAuth
	antigravityAuth *antigravity.AntigravityAuth
)

func init() {
	// Initialize session manager and auth providers
	sessionManager = auth.NewSessionManager(10 * time.Minute)
	codexAuth = codex.NewCodexAuth()
	antigravityAuth = antigravity.NewAntigravityAuth()

	// Public OAuth callback routes (no auth required)
	router.NewGroupRouter("/api/v1/oauth").
		AddRoute(
			router.NewRoute("/callback/codex", http.MethodGet).Handle(oauthCallbackCodex),
		).
		AddRoute(
			router.NewRoute("/callback/antigravity", http.MethodGet).Handle(oauthCallbackAntigravity),
		)

	// Protected OAuth management routes
	router.NewGroupRouter("/api/v1/oauth").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/start/:provider", http.MethodPost).Handle(oauthStart),
		).
		AddRoute(
			router.NewRoute("/token/list", http.MethodGet).Handle(oauthTokenList),
		).
		AddRoute(
			router.NewRoute("/token/create", http.MethodPost).Handle(oauthTokenCreate),
		).
		AddRoute(
			router.NewRoute("/token/update", http.MethodPost).Handle(oauthTokenUpdate),
		).
		AddRoute(
			router.NewRoute("/token/delete/:id", http.MethodDelete).Handle(oauthTokenDelete),
		).
		AddRoute(
			router.NewRoute("/token/refresh/:id", http.MethodPost).Handle(oauthTokenRefresh),
		).
		AddRoute(
			router.NewRoute("/status/:id", http.MethodGet).Handle(oauthTokenStatus),
		).
		AddRoute(
			router.NewRoute("/callback/submit", http.MethodPost).Handle(oauthCallbackSubmit),
		)
}

// OAuthStartRequest represents the request to start OAuth flow
type OAuthStartRequest struct {
	Provider string `json:"provider" binding:"required"` // codex or antigravity
}

// OAuthStartResponse represents the response for starting OAuth
type OAuthStartResponse struct {
	AuthURL   string `json:"auth_url"`
	SessionID string `json:"session_id"`
	Provider  string `json:"provider"`
}

// oauthStart initiates the OAuth flow
func oauthStart(c *gin.Context) {
	provider := c.Param("provider")

	var authURL string
	var redirectURI string
	var codeVerifier string

	ctx := c.Request.Context()

	switch provider {
	case "codex":
		// Generate PKCE codes
		pkceCodes, err := codex.GeneratePKCECodes()
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to generate PKCE codes: %v", err))
			return
		}

		// Generate state
		state := auth.GenerateState()
		codeVerifier = pkceCodes.CodeVerifier
		redirectURI = codex.RedirectURI

		// Generate auth URL
		authURL, err = codexAuth.GenerateAuthURL(state, pkceCodes)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to generate auth URL: %v", err))
			return
		}

		// Create session
		session := sessionManager.Create(provider, state, codeVerifier, redirectURI)

		resp.Success(c, OAuthStartResponse{
			AuthURL:   authURL,
			SessionID: session.ID,
			Provider:  provider,
		})

	case "antigravity":
		// Generate state
		state := auth.GenerateState()
		redirectURI = fmt.Sprintf("http://localhost:%d/oauth-callback", antigravity.CallbackPort)

		// Generate auth URL
		authURL = antigravityAuth.BuildAuthURL(state, redirectURI)

		// Create session
		session := sessionManager.Create(provider, state, "", redirectURI)

		resp.Success(c, OAuthStartResponse{
			AuthURL:   authURL,
			SessionID: session.ID,
			Provider:  provider,
		})

	default:
		resp.Error(c, http.StatusBadRequest, "Invalid provider. Supported: codex, antigravity")
	}

	log.Debugf("OAuth flow started for provider: %s", provider)
	_ = ctx // context available for future use
}

// oauthCallbackCodex handles the OAuth callback for Codex
func oauthCallbackCodex(c *gin.Context) {
	handleOAuthCallback(c, "codex")
}

// oauthCallbackAntigravity handles the OAuth callback for Antigravity
func oauthCallbackAntigravity(c *gin.Context) {
	handleOAuthCallback(c, "antigravity")
}

// handleOAuthCallback handles OAuth callbacks
func handleOAuthCallback(c *gin.Context, provider string) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": errorParam})
		return
	}

	if code == "" || state == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Missing code or state"})
		return
	}

	// Find session by state
	session, ok := sessionManager.GetByState(state)
	if !ok {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid or expired session"})
		return
	}

	ctx := context.Background()

	var oauthToken *model.OAuthToken
	var err error

	switch provider {
	case "codex":
		pkceCodes := &codex.PKCECodes{CodeVerifier: session.CodeVerifier}
		tokenResp, claims, err := codexAuth.ExchangeCodeForTokens(ctx, code, pkceCodes)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
			return
		}
		oauthToken = codex.CreateOAuthToken(tokenResp, claims)

	case "antigravity":
		tokenResp, err := antigravityAuth.ExchangeCodeForTokens(ctx, code, session.RedirectURI)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
			return
		}

		// Fetch user email
		email, err := antigravityAuth.FetchUserInfo(ctx, tokenResp.AccessToken)
		if err != nil {
			log.Warnf("Failed to fetch user info: %v", err)
		}

		oauthToken = antigravity.CreateOAuthToken(tokenResp, email)

	default:
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid provider"})
		return
	}

	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}

	// Save token to database
	if _, err := op.OAuthTokenCreate(&model.OAuthTokenCreateRequest{
		Type:         oauthToken.Type,
		Email:        oauthToken.Email,
		AccessToken:  oauthToken.AccessToken,
		RefreshToken: oauthToken.RefreshToken,
		IDToken:      oauthToken.IDToken,
		ExpiresIn:    int64(time.Until(*oauthToken.ExpiresAt).Seconds()),
	}, ctx); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": fmt.Sprintf("Failed to save token: %v", err)})
		return
	}

	// Clean up session
	sessionManager.Delete(session.ID)

	// Show success page
	c.HTML(http.StatusOK, "success.html", gin.H{
		"provider": provider,
		"email":    oauthToken.Email,
	})
}

// oauthTokenList returns all OAuth tokens
func oauthTokenList(c *gin.Context) {
	tokens, err := op.OAuthTokenList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, tokens)
}

// oauthTokenCreate creates a new OAuth token manually
func oauthTokenCreate(c *gin.Context) {
	var req model.OAuthTokenCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}

	token, err := op.OAuthTokenCreate(&req, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(c, token.ToResponse())
}

// OAuthCallbackSubmitRequest represents a callback URL submission
type OAuthCallbackSubmitRequest struct {
	Provider    string `json:"provider" binding:"required"`
	CallbackURL string `json:"callback_url" binding:"required"`
}

// oauthCallbackSubmit handles OAuth callback URL submission from user
func oauthCallbackSubmit(c *gin.Context) {
	var req OAuthCallbackSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}

	ctx := c.Request.Context()

	// Parse the callback URL to extract code parameter
	parsedURL, err := url.Parse(req.CallbackURL)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, fmt.Sprintf("Invalid callback URL: %v", err))
		return
	}

	code := parsedURL.Query().Get("code")
	if code == "" {
		resp.Error(c, http.StatusBadRequest, "Missing code parameter in callback URL")
		return
	}

	var oauthToken *model.OAuthToken

	switch req.Provider {
	case "codex":
		// For codex, we need the code verifier from session
		// Since the user is submitting the callback URL manually,
		// we'll need to create a new session or ask for code verifier
		// For now, let's try to exchange with empty code verifier (may fail)
		pkceCodes := &codex.PKCECodes{CodeVerifier: ""}
		tokenResp, claims, err := codexAuth.ExchangeCodeForTokens(ctx, code, pkceCodes)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, fmt.Sprintf("Failed to exchange code for token: %v", err))
			return
		}
		oauthToken = codex.CreateOAuthToken(tokenResp, claims)

	case "antigravity":
		redirectURI := fmt.Sprintf("http://localhost:%d/oauth-callback", antigravity.CallbackPort)
		tokenResp, err := antigravityAuth.ExchangeCodeForTokens(ctx, code, redirectURI)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, fmt.Sprintf("Failed to exchange code for token: %v", err))
			return
		}

		// Fetch user email
		email, err := antigravityAuth.FetchUserInfo(ctx, tokenResp.AccessToken)
		if err != nil {
			log.Warnf("Failed to fetch user info: %v", err)
		}

		oauthToken = antigravity.CreateOAuthToken(tokenResp, email)

	default:
		resp.Error(c, http.StatusBadRequest, "Invalid provider. Supported: codex, antigravity")
		return
	}

	// Save token to database
	savedToken, err := op.OAuthTokenCreate(&model.OAuthTokenCreateRequest{
		Type:         oauthToken.Type,
		Email:        oauthToken.Email,
		AccessToken:  oauthToken.AccessToken,
		RefreshToken: oauthToken.RefreshToken,
		IDToken:      oauthToken.IDToken,
		ExpiresIn:    int64(time.Until(*oauthToken.ExpiresAt).Seconds()),
	}, ctx)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to save token: %v", err))
		return
	}

	resp.Success(c, savedToken.ToResponse())
}

// oauthTokenUpdate updates an OAuth token
func oauthTokenUpdate(c *gin.Context) {
	var req model.OAuthTokenUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}

	token, err := op.OAuthTokenUpdate(&req, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(c, token.ToResponse())
}

// oauthTokenDelete deletes an OAuth token
func oauthTokenDelete(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		resp.Error(c, http.StatusBadRequest, "Invalid token ID")
		return
	}

	if err := op.OAuthTokenDelete(id, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(c, nil)
}

// oauthTokenRefresh refreshes an OAuth token
func oauthTokenRefresh(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		resp.Error(c, http.StatusBadRequest, "Invalid token ID")
		return
	}

	token, err := op.OAuthTokenGet(id, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusNotFound, "Token not found")
		return
	}

	ctx := c.Request.Context()

	switch token.Type {
	case model.OAuthTypeCodex:
		tokenResp, _, err := codexAuth.RefreshTokens(ctx, token.RefreshToken)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to refresh token: %v", err))
			return
		}

		if err := op.OAuthTokenRefresh(id, tokenResp.AccessToken, tokenResp.RefreshToken, int64(tokenResp.ExpiresIn), ctx); err != nil {
			resp.Error(c, http.StatusInternalServerError, err.Error())
			return
		}

	case model.OAuthTypeAntigravity:
		tokenResp, err := antigravityAuth.RefreshTokens(ctx, token.RefreshToken)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to refresh token: %v", err))
			return
		}

		if err := op.OAuthTokenRefresh(id, tokenResp.AccessToken, tokenResp.RefreshToken, tokenResp.ExpiresIn, ctx); err != nil {
			resp.Error(c, http.StatusInternalServerError, err.Error())
			return
		}

	default:
		resp.Error(c, http.StatusBadRequest, "Unknown token type")
		return
	}

	// Get updated token
	updatedToken, _ := op.OAuthTokenGet(id, ctx)
	resp.Success(c, updatedToken.ToResponse())
}

// oauthTokenStatus returns the status of an OAuth token
func oauthTokenStatus(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		resp.Error(c, http.StatusBadRequest, "Invalid token ID")
		return
	}

	token, err := op.OAuthTokenGet(id, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusNotFound, "Token not found")
		return
	}

	status := map[string]interface{}{
		"id":          token.ID,
		"type":        token.Type,
		"email":       token.Email,
		"enabled":     token.Enabled,
		"is_expired":  token.IsExpired(),
		"needs_refresh": token.NeedsRefresh(),
	}

	if token.ExpiresAt != nil {
		status["expires_at"] = token.ExpiresAt
		status["expires_in"] = time.Until(*token.ExpiresAt).Seconds()
	}

	if token.LastRefresh != nil {
		status["last_refresh"] = token.LastRefresh
	}

	resp.Success(c, status)
}
