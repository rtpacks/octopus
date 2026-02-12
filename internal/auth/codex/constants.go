package codex

// OAuth configuration constants for OpenAI Codex
const (
	AuthURL     = "https://auth.openai.com/oauth/authorize"
	TokenURL    = "https://auth.openai.com/oauth/token"
	ClientID    = "app_EMoamEEZ73f0CkXaXp7hrann"
	RedirectURI = "http://localhost:1455/auth/callback"
	CallbackPort = 1455
)

// Scopes for Codex OAuth
var Scopes = []string{"openid", "email", "profile", "offline_access"}
