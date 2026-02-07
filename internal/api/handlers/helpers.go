package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey    contextKey = "user_id"
	userEmailKey contextKey = "user_email"
	isAdminKey   contextKey = "is_admin"
)

type errorResponse struct {
	Error string `json:"error"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, errorResponse{Error: message})
}

func GetUserIDFromContext(ctx context.Context) uuid.UUID {
	if userID, ok := ctx.Value(userIDKey).(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}

func GetUserEmailFromContext(ctx context.Context) string {
	if email, ok := ctx.Value(userEmailKey).(string); ok {
		return email
	}
	return ""
}

func GetIsAdminFromContext(ctx context.Context) bool {
	if isAdmin, ok := ctx.Value(isAdminKey).(bool); ok {
		return isAdmin
	}
	return false
}

func SetUserContext(ctx context.Context, userID uuid.UUID, email string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, userEmailKey, email)
	ctx = context.WithValue(ctx, isAdminKey, isAdmin)
	return ctx
}
