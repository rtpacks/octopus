package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/utils/log"
)

// OAuthServer handles the local HTTP server for OAuth callbacks
type OAuthServer struct {
	server     *http.Server
	port       int
	resultChan chan *OAuthResult
	errorChan  chan error
	mu         sync.Mutex
	running    bool
}

// OAuthResult contains the result of the OAuth callback
type OAuthResult struct {
	Code  string
	State string
	Error string
}

// NewOAuthServer creates a new OAuth callback server
func NewOAuthServer(port int) *OAuthServer {
	return &OAuthServer{
		port:       port,
		resultChan: make(chan *OAuthResult, 1),
		errorChan:  make(chan error, 1),
	}
}

// Start starts the OAuth callback server
func (s *OAuthServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	if !s.isPortAvailable() {
		return fmt.Errorf("port %d is already in use", s.port)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", s.handleCallback)
	mux.HandleFunc("/oauth-callback", s.handleCallback) // Alternative path for antigravity
	mux.HandleFunc("/success", s.handleSuccess)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	s.running = true

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.errorChan <- fmt.Errorf("server failed to start: %w", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	return nil
}

// Stop gracefully stops the OAuth callback server
func (s *OAuthServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.server == nil {
		return nil
	}

	log.Debugf("Stopping OAuth callback server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.server.Shutdown(shutdownCtx)
	s.running = false
	s.server = nil

	return err
}

// WaitForCallback waits for the OAuth callback with a timeout
func (s *OAuthServer) WaitForCallback(timeout time.Duration) (*OAuthResult, error) {
	select {
	case result := <-s.resultChan:
		return result, nil
	case err := <-s.errorChan:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for OAuth callback")
	}
}

// handleCallback handles the OAuth callback endpoint
func (s *OAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Received OAuth callback")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	code := query.Get("code")
	state := query.Get("state")
	errorParam := query.Get("error")

	if errorParam != "" {
		log.Errorf("OAuth error received: %s", errorParam)
		result := &OAuthResult{Error: errorParam}
		s.sendResult(result)
		http.Error(w, fmt.Sprintf("OAuth error: %s", errorParam), http.StatusBadRequest)
		return
	}

	if code == "" {
		log.Errorf("No authorization code received")
		result := &OAuthResult{Error: "no_code"}
		s.sendResult(result)
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	if state == "" {
		log.Errorf("No state parameter received")
		result := &OAuthResult{Error: "no_state"}
		s.sendResult(result)
		http.Error(w, "No state parameter received", http.StatusBadRequest)
		return
	}

	result := &OAuthResult{Code: code, State: state}
	s.sendResult(result)
	http.Redirect(w, r, "/success", http.StatusFound)
}

// handleSuccess handles the success page endpoint
func (s *OAuthServer) handleSuccess(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Serving success page")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	html := generateSuccessHTML()
	_, _ = w.Write([]byte(html))
}

// sendResult sends the OAuth result to the waiting channel
func (s *OAuthServer) sendResult(result *OAuthResult) {
	select {
	case s.resultChan <- result:
		log.Debugf("OAuth result sent to channel")
	default:
		log.Warnf("OAuth result channel is full, result dropped")
	}
}

// isPortAvailable checks if the specified port is available
func (s *OAuthServer) isPortAvailable() bool {
	addr := fmt.Sprintf(":%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

// IsRunning returns whether the server is currently running
func (s *OAuthServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// GetPort returns the server port
func (s *OAuthServer) GetPort() int {
	return s.port
}

// generateSuccessHTML creates the HTML content for the success page
func generateSuccessHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Successful - Octopus</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            display: flex; justify-content: center; align-items: center;
            min-height: 100vh; margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 1rem;
        }
        .container {
            text-align: center; background: white; padding: 2.5rem;
            border-radius: 12px; box-shadow: 0 10px 25px rgba(0,0,0,0.1);
            max-width: 480px; width: 100%;
            animation: slideIn 0.3s ease-out;
        }
        @keyframes slideIn {
            from { opacity: 0; transform: translateY(-20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .success-icon {
            width: 64px; height: 64px; margin: 0 auto 1.5rem;
            background: #10b981; border-radius: 50%;
            display: flex; align-items: center; justify-content: center;
            color: white; font-size: 2rem; font-weight: bold;
        }
        h1 { color: #1f2937; margin-bottom: 1rem; font-size: 1.75rem; font-weight: 600; }
        .subtitle { color: #6b7280; margin-bottom: 1.5rem; font-size: 1rem; line-height: 1.5; }
        .button {
            padding: 0.75rem 1.5rem; border-radius: 8px;
            font-size: 0.875rem; font-weight: 500;
            background: #3b82f6; color: white; border: none;
            cursor: pointer; transition: all 0.2s;
        }
        .button:hover { background: #2563eb; transform: translateY(-1px); }
        .countdown { color: #9ca3af; font-size: 0.75rem; margin-top: 1rem; }
        .footer {
            margin-top: 2rem; padding-top: 1.5rem;
            border-top: 1px solid #e5e7eb; color: #9ca3af; font-size: 0.75rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="success-icon">✓</div>
        <h1>Authentication Successful!</h1>
        <p class="subtitle">You have successfully authenticated. You can now close this window and return to Octopus.</p>
        <button class="button" onclick="window.close()">Close Window</button>
        <div class="countdown">This window will close automatically in <span id="countdown">10</span> seconds</div>
        <div class="footer"><p>Powered by Octopus</p></div>
    </div>
    <script>
        let countdown = 10;
        const countdownElement = document.getElementById('countdown');
        const timer = setInterval(() => {
            countdown--;
            countdownElement.textContent = countdown;
            if (countdown <= 0) { clearInterval(timer); window.close(); }
        }, 1000);
        document.addEventListener('keydown', (e) => { if (e.key === 'Escape') window.close(); });
    </script>
</body>
</html>`
}

// GetCallbackURL returns the callback URL for the server
func (s *OAuthServer) GetCallbackURL() string {
	return fmt.Sprintf("http://localhost:%d/auth/callback", s.port)
}

// GetAlternativeCallbackURL returns the alternative callback URL
func (s *OAuthServer) GetAlternativeCallbackURL() string {
	return fmt.Sprintf("http://localhost:%d/oauth-callback", s.port)
}

// ParseProviderFromPath extracts provider info from the callback path
func ParseProviderFromPath(path string) string {
	if strings.Contains(path, "codex") {
		return "codex"
	}
	if strings.Contains(path, "antigravity") {
		return "antigravity"
	}
	return ""
}
