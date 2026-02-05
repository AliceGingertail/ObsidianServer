package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/yourusername/ObsidianServer/internal/services"
	domerrors "github.com/yourusername/ObsidianServer/internal/domain/errors"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Username, email and password are required")
		return
	}

	if len(req.Username) < 3 {
		respondError(w, http.StatusBadRequest, "Username must be at least 3 characters")
		return
	}

	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	resp, err := h.authService.Register(r.Context(), &services.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		if err == domerrors.ErrUserExists {
			respondError(w, http.StatusConflict, "User already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to register user")
		return
	}

	respondJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	deviceInfo := r.Header.Get("User-Agent")

	resp, err := h.authService.Login(r.Context(), &services.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	}, deviceInfo)

	if err != nil {
		if err == domerrors.ErrInvalidCredentials {
			respondError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to login")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "Refresh token is required")
		return
	}

	resp, err := h.authService.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		if err == domerrors.ErrInvalidToken {
			respondError(w, http.StatusUnauthorized, "Invalid refresh token")
			return
		}
		if err == domerrors.ErrTokenExpired {
			respondError(w, http.StatusUnauthorized, "Refresh token expired")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to refresh tokens")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	if err := h.authService.Logout(r.Context(), userID, req.RefreshToken); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to logout")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())

	if err := h.authService.LogoutAll(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to logout from all devices")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Logged out from all devices"})
}
