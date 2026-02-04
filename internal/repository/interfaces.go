package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourusername/ObsidianServer/internal/domain/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetAll(ctx context.Context) ([]*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int, error)
}

type PeerRepository interface {
	Create(ctx context.Context, peer *models.Peer) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Peer, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Peer, error)
	GetByPublicKey(ctx context.Context, publicKey string) (*models.Peer, error)
	GetAll(ctx context.Context) ([]*models.Peer, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	Count(ctx context.Context) (int, error)
	CountActive(ctx context.Context) (int, error)
	Update(ctx context.Context, peer *models.Peer) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetLastIPAddress(ctx context.Context, protocol models.Protocol) (string, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.RefreshToken, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}
