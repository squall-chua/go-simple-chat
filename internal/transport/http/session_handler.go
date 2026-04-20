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
	Cert string `json:"cert"`
}

type sessionResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

func (h *SessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*") // For dev; restrict in prod if needed
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req sessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Cert == "" {
		http.Error(w, "Certificate is required", http.StatusBadRequest)
		return
	}

	token, userID, username, err := h.sessionService.IssueToken(r.Context(), []byte(req.Cert))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessionResponse{
		Token:    token,
		UserID:   userID,
		Username: username,
	})
}
