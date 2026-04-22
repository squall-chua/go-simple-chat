package http

import (
	"encoding/json"
	"go-simple-chat/internal/service"
	"net/http"
)

type SessionHandler struct {
	sessionService *service.SessionService
}

func NewSessionHandler(sessionService *service.SessionService) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
	}
}

type sessionRequest struct {
	Cert      string `json:"cert"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}

type sessionResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

type challengeResponse struct {
	Nonce string `json:"nonce"`
}

func (h *SessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/session/challenge":
		h.handleChallenge(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/session":
		h.handleLogin(w, r)
	case r.Method == http.MethodDelete && r.URL.Path == "/api/session":
		h.handleLogout(w, r)
	default:
		http.Error(w, "Method not allowed or path not found", http.StatusMethodNotAllowed)
	}
}

func (h *SessionHandler) handleChallenge(w http.ResponseWriter, r *http.Request) {
	nonce, err := h.sessionService.CreateChallenge(r.Context())
	if err != nil {
		http.Error(w, "Failed to create challenge", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(challengeResponse{Nonce: nonce})
}

func (h *SessionHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req sessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Cert == "" || req.Nonce == "" || req.Signature == "" {
		http.Error(w, "Certificate, Nonce, and Signature are required", http.StatusBadRequest)
		return
	}

	token, userID, username, err := h.sessionService.IssueToken(r.Context(), []byte(req.Cert), req.Nonce, req.Signature)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Set secure cookie for web client
	http.SetCookie(w, &http.Cookie{
		Name:     "x-session-token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   86400, // 24h
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessionResponse{
		Token:    token,
		UserID:   userID,
		Username: username,
	})
}

func (h *SessionHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "x-session-token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}
