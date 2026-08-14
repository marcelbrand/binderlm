package drive

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

// Default OAuth2 credentials for binderlm desktop CLI.
// Users can override these via environment variables GDRIVE_OAUTH_CLIENT_ID and GDRIVE_OAUTH_CLIENT_SECRET.
var (
	DefaultClientID     = "981245037169-binderlm-desktop-app.apps.googleusercontent.com"
	DefaultClientSecret = "GOCSPX-binderlm-desktop-secret"
)

const (
	UserInfoEndpoint = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// TokenStorage handles reading, writing, and deleting cached OAuth tokens.
type TokenStorage struct {
	customPath string
}

// NewTokenStorage creates a new token storage manager.
func NewTokenStorage(customPath ...string) *TokenStorage {
	var path string
	if len(customPath) > 0 && customPath[0] != "" {
		path = customPath[0]
	}
	return &TokenStorage{customPath: path}
}

// TokenPath returns the absolute path where the OAuth token is stored.
func (s *TokenStorage) TokenPath() (string, error) {
	if s.customPath != "" {
		return s.customPath, nil
	}
	if envPath := os.Getenv("BINDERLM_TOKEN_FILE"); envPath != "" {
		return envPath, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, hErr := os.UserHomeDir()
		if hErr != nil {
			return "", fmt.Errorf("could not determine user configuration directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "binderlm", "token.json"), nil
}

// Save stores the OAuth token in secure JSON format.
func (s *TokenStorage) Save(token *oauth2.Token) error {
	path, err := s.TokenPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize token: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write token to %s: %w", path, err)
	}

	return nil
}

// Load reads and unmarshals the cached token.
func (s *TokenStorage) Load() (*oauth2.Token, error) {
	path, err := s.TokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("corrupted token file at %s: %w", path, err)
	}

	return &token, nil
}

// Delete removes the cached token file.
func (s *TokenStorage) Delete() error {
	path, err := s.TokenPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token at %s: %w", path, err)
	}

	return nil
}

// Exists checks whether a cached token file exists on disk.
func (s *TokenStorage) Exists() bool {
	path, err := s.TokenPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// OAuthManager manages interactive OAuth authentication.
type OAuthManager struct {
	Config  *oauth2.Config
	Storage *TokenStorage
}

// NewOAuthManager creates a new OAuth manager with configured credentials.
func NewOAuthManager(clientID, clientSecret string, storage *TokenStorage) *OAuthManager {
	if clientID == "" {
		clientID = os.Getenv("GDRIVE_OAUTH_CLIENT_ID")
	}
	if clientID == "" {
		clientID = DefaultClientID
	}

	if clientSecret == "" {
		clientSecret = os.Getenv("GDRIVE_OAUTH_CLIENT_SECRET")
	}
	if clientSecret == "" {
		clientSecret = DefaultClientSecret
	}

	if storage == nil {
		storage = NewTokenStorage()
	}

	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			drive.DriveScope,
			"https://www.googleapis.com/auth/userinfo.email",
		},
	}

	return &OAuthManager{
		Config:  conf,
		Storage: storage,
	}
}

// AutoSavingTokenSource wraps oauth2.TokenSource to persist refreshed tokens automatically.
type AutoSavingTokenSource struct {
	source  oauth2.TokenSource
	storage *TokenStorage
	mu      sync.Mutex
}

func (s *AutoSavingTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tok, err := s.source.Token()
	if err != nil {
		return nil, err
	}

	// Persist updated token to disk
	_ = s.storage.Save(tok)
	return tok, nil
}

// GetTokenSource returns an auto-refreshing TokenSource backed by the cached token.
func (m *OAuthManager) GetTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	tok, err := m.Storage.Load()
	if err != nil {
		return nil, err
	}

	ts := m.Config.TokenSource(ctx, tok)
	return &AutoSavingTokenSource{
		source:  ts,
		storage: m.Storage,
	}, nil
}

// FetchUserInfo retrieves user email associated with the token.
func FetchUserInfo(ctx context.Context, ts oauth2.TokenSource) (string, error) {
	client := oauth2.NewClient(ctx, ts)
	resp, err := client.Get(UserInfoEndpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("userinfo API returned status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", err
	}

	return userInfo.Email, nil
}

// LoginOptions defines options for interactive login.
type LoginOptions struct {
	Port        int
	OpenBrowser bool
	Timeout     time.Duration
}

// LoginResult contains the result of a successful login.
type LoginResult struct {
	Email string
	Token *oauth2.Token
}

// Login performs an interactive OAuth2 login flow via a local web server.
func (m *OAuthManager) Login(ctx context.Context, opts LoginOptions) (*LoginResult, error) {
	if opts.Port <= 0 {
		opts.Port = 8085
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 3 * time.Minute
	}

	// Generate random state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random OAuth state: %w", err)
	}
	expectedState := hex.EncodeToString(stateBytes)

	// Bind to local loopback port
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", opts.Port))
	if err != nil {
		// Fallback to random available port if requested port is busy
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, fmt.Errorf("failed to start local OAuth listener: %w", err)
		}
	}
	defer listener.Close()

	actualPort := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/oauth2callback", actualPort)
	m.Config.RedirectURL = redirectURL

	authURL := m.Config.AuthCodeURL(
		expectedState,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if errMsg := query.Get("error"); errMsg != "" {
			http.Error(w, fmt.Sprintf("Authentication failed: %s", errMsg), http.StatusBadRequest)
			errChan <- fmt.Errorf("OAuth error received: %s", errMsg)
			return
		}

		state := query.Get("state")
		if state != expectedState {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errChan <- errors.New("state mismatch in OAuth callback")
			return
		}

		code := query.Get("code")
		if code == "" {
			http.Error(w, "No code provided in callback", http.StatusBadRequest)
			errChan <- errors.New("no authorization code received in callback")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>binderlm Authentication Successful</title></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #0f172a; color: #f8fafc;">
  <div style="background: #1e293b; padding: 40px 60px; border-radius: 12px; box-shadow: 0 10px 25px rgba(0,0,0,0.5); text-align: center; border: 1px solid #334155;">
    <h1 style="color: #38bdf8; margin-top: 0;">Authentication Successful</h1>
    <p style="color: #94a3b8; font-size: 16px;">You have successfully connected binderlm to your Google Drive account.</p>
    <p style="color: #64748b; font-size: 14px;">You may now safely close this browser window and return to your terminal.</p>
  </div>
</body>
</html>`)
		codeChan <- code
	})

	server := &http.Server{Handler: mux}
	go func() {
		if sErr := server.Serve(listener); sErr != nil && !errors.Is(sErr, http.ErrServerClosed) {
			errChan <- sErr
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Open browser if requested
	if opts.OpenBrowser {
		_ = openBrowserURL(authURL)
	}

	fmt.Println("Please open the following authorization URL in your browser to complete login:")
	fmt.Println()
	fmt.Printf("  %s\n\n", authURL)
	fmt.Println("Waiting for authorization in browser...")

	// Wait for code or timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(opts.Timeout):
		return nil, errors.New("authentication timed out waiting for browser callback")
	case err := <-errChan:
		return nil, err
	case code := <-codeChan:
		// Exchange code for token
		token, err := m.Config.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange authorization code for token: %w", err)
		}

		// Save token to disk
		if err := m.Storage.Save(token); err != nil {
			return nil, fmt.Errorf("failed to save OAuth token: %w", err)
		}

		// Fetch user email
		ts := m.Config.TokenSource(ctx, token)
		email, _ := FetchUserInfo(ctx, ts)

		return &LoginResult{
			Email: email,
			Token: token,
		}, nil
	}
}

func openBrowserURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		// linux / bsd
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
