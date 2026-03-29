package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type config struct {
	port         string
	issuerURL    string
	clientID     string
	clientSecret string
	defaultUser  userProfile
}

type userProfile struct {
	Sub               string `json:"sub"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	Role              string `json:"role"`
	PreferredUsername string `json:"preferred_username"`
}

type authCode struct {
	Profile     userProfile
	RedirectURI string
	ExpiresAt   time.Time
}

type accessToken struct {
	Profile   userProfile
	ExpiresAt time.Time
}

type server struct {
	cfg         config
	privateKey  *rsa.PrivateKey
	publicKey   rsa.PublicKey
	codes       map[string]authCode
	accessToken map[string]accessToken
	mu          sync.Mutex
}

func main() {
	cfg := config{
		port:         getEnv("MOCK_OIDC_PORT", "8090"),
		issuerURL:    getEnv("MOCK_OIDC_ISSUER_URL", "http://localhost:19090"),
		clientID:     getEnv("MOCK_OIDC_CLIENT_ID", "credit-flow-web"),
		clientSecret: getEnv("MOCK_OIDC_CLIENT_SECRET", "credit-flow-local-secret"),
		defaultUser: userProfile{
			Sub:               getEnv("MOCK_OIDC_DEFAULT_SUB", "operator-001"),
			Name:              getEnv("MOCK_OIDC_DEFAULT_NAME", "Local Operator"),
			Email:             getEnv("MOCK_OIDC_DEFAULT_EMAIL", "operator@local.test"),
			Role:              getEnv("MOCK_OIDC_DEFAULT_ROLE", "operations"),
			PreferredUsername: getEnv("MOCK_OIDC_DEFAULT_USERNAME", "local.operator"),
		},
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("generate rsa key: %v", err)
	}

	srv := &server{
		cfg:         cfg,
		privateKey:  privateKey,
		publicKey:   privateKey.PublicKey,
		codes:       make(map[string]authCode),
		accessToken: make(map[string]accessToken),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", srv.handleHealthz)
	mux.HandleFunc("/.well-known/openid-configuration", srv.handleDiscovery)
	mux.HandleFunc("/jwks", srv.handleJWKS)
	mux.HandleFunc("/authorize", srv.handleAuthorize)
	mux.HandleFunc("/token", srv.handleToken)
	mux.HandleFunc("/userinfo", srv.handleUserInfo)

	httpServer := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("mock oidc listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}

func (s *server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleDiscovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                 s.cfg.issuerURL,
		"authorization_endpoint": s.issuerURL("/authorize"),
		"token_endpoint":         s.issuerURL("/token"),
		"userinfo_endpoint":      s.issuerURL("/userinfo"),
		"jwks_uri":               s.issuerURL("/jwks"),
		"response_types_supported": []string{
			"code",
		},
		"subject_types_supported": []string{
			"public",
		},
		"id_token_signing_alg_values_supported": []string{
			"RS256",
		},
	})
}

func (s *server) handleJWKS(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": "mock-key-1",
				"n":   base64.RawURLEncoding.EncodeToString(s.publicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(s.publicKey.E)).Bytes()),
			},
		},
	})
}

func (s *server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	redirectURI := strings.TrimSpace(query.Get("redirect_uri"))
	state := strings.TrimSpace(query.Get("state"))
	clientID := strings.TrimSpace(query.Get("client_id"))

	if redirectURI == "" || clientID == "" || clientID != s.cfg.clientID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	code, err := randomToken(24)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	s.mu.Lock()
	s.codes[code] = authCode{
		Profile:     s.cfg.defaultUser,
		RedirectURI: redirectURI,
		ExpiresAt:   time.Now().UTC().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	target, err := url.Parse(redirectURI)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_redirect_uri"})
		return
	}
	params := target.Query()
	params.Set("code", code)
	if state != "" {
		params.Set("state", state)
	}
	target.RawQuery = params.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func (s *server) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	if r.PostForm.Get("grant_type") != "authorization_code" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
		return
	}

	if strings.TrimSpace(r.PostForm.Get("client_id")) != s.cfg.clientID {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
		return
	}
	if s.cfg.clientSecret != "" && strings.TrimSpace(r.PostForm.Get("client_secret")) != s.cfg.clientSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
		return
	}

	code := strings.TrimSpace(r.PostForm.Get("code"))
	redirectURI := strings.TrimSpace(r.PostForm.Get("redirect_uri"))

	s.mu.Lock()
	stored, ok := s.codes[code]
	if ok {
		delete(s.codes, code)
	}
	s.mu.Unlock()

	if !ok || time.Now().UTC().After(stored.ExpiresAt) || redirectURI != stored.RedirectURI {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}

	accessValue, err := randomToken(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	s.mu.Lock()
	s.accessToken[accessValue] = accessToken{
		Profile:   stored.Profile,
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
	}
	s.mu.Unlock()

	idToken, err := s.signIDToken(stored.Profile)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessValue,
		"id_token":     idToken,
		"token_type":   "Bearer",
		"expires_in":   900,
	})
}

func (s *server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		return
	}

	s.mu.Lock()
	stored, ok := s.accessToken[token]
	s.mu.Unlock()

	if !ok || time.Now().UTC().After(stored.ExpiresAt) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		return
	}

	writeJSON(w, http.StatusOK, stored.Profile)
}

func (s *server) signIDToken(profile userProfile) (string, error) {
	header := map[string]string{
		"alg": "RS256",
		"kid": "mock-key-1",
		"typ": "JWT",
	}
	now := time.Now().UTC()
	claims := map[string]any{
		"iss":                s.cfg.issuerURL,
		"aud":                s.cfg.clientID,
		"sub":                profile.Sub,
		"name":               profile.Name,
		"email":              profile.Email,
		"role":               profile.Role,
		"preferred_username": profile.PreferredUsername,
		"iat":                now.Unix(),
		"exp":                now.Add(15 * time.Minute).Unix(),
	}

	headerRaw, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsRaw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerRaw) + "." + base64.RawURLEncoding.EncodeToString(claimsRaw)
	sum := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}

	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *server) issuerURL(path string) string {
	return strings.TrimRight(s.cfg.issuerURL, "/") + path
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func randomToken(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
