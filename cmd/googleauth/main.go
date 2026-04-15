package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendarapi "google.golang.org/api/calendar/v3"
	"linebot-backend/internal/infra"
)

func main() {
	cfg := infra.LoadConfigFromEnv()
	if cfg.GoogleOAuthCredentialsFile == "" {
		log.Fatal("LINEBOT_GOOGLE_OAUTH_CREDENTIALS_FILE is required")
	}
	if cfg.GoogleOAuthTokenFile == "" {
		log.Fatal("LINEBOT_GOOGLE_OAUTH_TOKEN_FILE is required")
	}

	credentials, err := os.ReadFile(cfg.GoogleOAuthCredentialsFile)
	if err != nil {
		log.Fatalf("Read credentials file failed: %v", err)
	}

	oauthConfig, err := google.ConfigFromJSON(credentials, calendarapi.CalendarEventsScope)
	if err != nil {
		log.Fatalf("Parse credentials file failed: %v", err)
	}

	token, err := runOAuthFlow(oauthConfig)
	if err != nil {
		log.Fatalf("OAuth flow failed: %v", err)
	}

	if err := writeToken(cfg.GoogleOAuthTokenFile, token); err != nil {
		log.Fatalf("Write token failed: %v", err)
	}

	log.Printf("Google OAuth token saved to %s", cfg.GoogleOAuthTokenFile)
}

func runOAuthFlow(oauthConfig *oauth2.Config) (*oauth2.Token, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen local callback: %w", err)
	}
	defer listener.Close()

	redirectURL := "http://" + listener.Addr().String() + "/callback"
	oauthConfig.RedirectURL = redirectURL

	state, err := randomState()
	if err != nil {
		return nil, err
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "invalid OAuth state", http.StatusBadRequest)
			errCh <- fmt.Errorf("invalid OAuth state")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing OAuth code", http.StatusBadRequest)
			errCh <- fmt.Errorf("missing OAuth code")
			return
		}

		_, _ = fmt.Fprintln(w, "Google Calendar authorization succeeded. You can close this window.")
		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Shutdown(context.Background())

	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	log.Printf("Open this URL in your browser and finish Google authorization:\n%s", authURL)
	log.Printf("Waiting for OAuth callback on %s", redirectURL)

	select {
	case code := <-codeCh:
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return oauthConfig.Exchange(ctx, code)
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("timeout waiting for OAuth callback")
	}
}

func randomState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate OAuth state: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func writeToken(path string, token *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(token)
}
