package middleware

import (
	"net/http"
	"strings"

	"github.com/yourusername/ObsidianServer/internal/api/handlers"
	"github.com/yourusername/ObsidianServer/pkg/jwt"
)

type AuthMiddleware struct {
	jwtManager *jwt.Manager
}

func NewAuthMiddleware(jwtManager *jwt.Manager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем токен из заголовка
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		// Проверяем формат "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		token := parts[1]

		// Валидируем токен
		claims, err := m.jwtManager.ValidateAccessToken(token)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Добавляем информацию о пользователе в контекст
		ctx := handlers.SetUserContext(r.Context(), claims.UserID, claims.Email, claims.IsAdmin)

		// Передаем управление следующему обработчику
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin := handlers.GetIsAdminFromContext(r.Context())
		if !isAdmin {
			respondError(w, http.StatusForbidden, "Admin access required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
