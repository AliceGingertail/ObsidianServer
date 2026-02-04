package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourusername/ObsidianServer/internal/domain/models"
	domerrors "github.com/yourusername/ObsidianServer/internal/domain/errors"
	"github.com/yourusername/ObsidianServer/internal/repository"
	"github.com/yourusername/ObsidianServer/pkg/crypto"
	"github.com/yourusername/ObsidianServer/pkg/jwt"
)

type AuthService struct {
	userRepo  repository.UserRepository
	tokenRepo repository.RefreshTokenRepository
	jwtManager *jwt.Manager
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	jwtManager *jwt.Manager,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtManager: jwtManager,
	}
}

type RegisterRequest struct {
	Username string
	Email    string
	Password string
}

type LoginRequest struct {
	Username string
	Password string
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *models.User `json:"user"`
}

func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// Проверяем, существует ли пользователь с таким email
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, domerrors.ErrUserExists
	}

	// Хешируем пароль
	passwordHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создаем пользователя
	user := models.NewUser(req.Username, req.Email, passwordHash)
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Генерируем токены
	return s.generateTokens(ctx, user, "")
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest, deviceInfo string) (*AuthResponse, error) {
	// Получаем пользователя по username
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		if err == domerrors.ErrUserNotFound {
			return nil, domerrors.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Проверяем пароль
	if !crypto.CheckPassword(req.Password, user.Password) {
		return nil, domerrors.ErrInvalidCredentials
	}

	// Проверяем активность пользователя
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Генерируем токены
	return s.generateTokens(ctx, user, deviceInfo)
}

func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Хешируем токен для поиска в БД
	tokenHash := crypto.HashToken(refreshToken)

	// Получаем токен из БД
	storedToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, domerrors.ErrInvalidToken
	}

	// Проверяем истечение
	if storedToken.IsExpired() {
		// Удаляем истекший токен
		_ = s.tokenRepo.Delete(ctx, storedToken.ID)
		return nil, domerrors.ErrTokenExpired
	}

	// Получаем пользователя
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Удаляем старый refresh token
	_ = s.tokenRepo.Delete(ctx, storedToken.ID)

	// Генерируем новые токены
	return s.generateTokens(ctx, user, storedToken.DeviceInfo)
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	tokenHash := crypto.HashToken(refreshToken)
	
	storedToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return err
	}

	if storedToken.UserID != userID {
		return domerrors.ErrUnauthorized
	}

	return s.tokenRepo.Delete(ctx, storedToken.ID)
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.tokenRepo.DeleteByUserID(ctx, userID)
}

func (s *AuthService) generateTokens(ctx context.Context, user *models.User, deviceInfo string) (*AuthResponse, error) {
	// Генерируем access token
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Генерируем refresh token
	refreshToken, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Сохраняем refresh token в БД
	tokenHash := crypto.HashToken(refreshToken)
	refreshTokenModel := models.NewRefreshToken(
		user.ID,
		tokenHash,
		deviceInfo,
		s.jwtManager.GetRefreshTTL(),
	)

	if err := s.tokenRepo.Create(ctx, refreshTokenModel); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}
