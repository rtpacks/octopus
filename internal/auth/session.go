package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// OAuthSession stores the state for an ongoing OAuth flow
type OAuthSession struct {
	ID           string
	Provider     string
	State        string
	CodeVerifier string
	RedirectURI  string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// SessionManager manages OAuth sessions with TTL
type SessionManager struct {
	sessions map[string]*OAuthSession
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(ttl time.Duration) *SessionManager {
	if ttl == 0 {
		ttl = 10 * time.Minute
	}
	sm := &SessionManager{
		sessions: make(map[string]*OAuthSession),
		ttl:      ttl,
	}
	go sm.cleanup()
	return sm
}

// Create creates a new OAuth session
func (sm *SessionManager) Create(provider, state, codeVerifier, redirectURI string) *OAuthSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	session := &OAuthSession{
		ID:           generateSessionID(),
		Provider:     provider,
		State:        state,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectURI,
		CreatedAt:    now,
		ExpiresAt:    now.Add(sm.ttl),
	}

	sm.sessions[session.ID] = session
	return session
}

// Get retrieves a session by ID
func (sm *SessionManager) Get(id string) (*OAuthSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[id]
	if !ok {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

// GetByState retrieves a session by state parameter
func (sm *SessionManager) GetByState(state string) (*OAuthSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, session := range sm.sessions {
		if session.State == state && !time.Now().After(session.ExpiresAt) {
			return session, true
		}
	}
	return nil, false
}

// Delete removes a session
func (sm *SessionManager) Delete(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, id)
}

// cleanup removes expired sessions periodically
func (sm *SessionManager) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateState generates a random state parameter for CSRF protection
func GenerateState() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
