package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/ObsidianServer/internal/domain/models"
	domerrors "github.com/yourusername/ObsidianServer/internal/domain/errors"
)

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, device_info, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.DeviceInfo,
		token.ExpiresAt,
		token.CreatedAt,
	)
	return err
}

func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, device_info, expires_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	
	token := &models.RefreshToken{}
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.DeviceInfo,
		&token.ExpiresAt,
		&token.CreatedAt,
	)
	
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domerrors.ErrInvalidToken
		}
		return nil, err
	}
	
	return token, nil
}

func (r *RefreshTokenRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, device_info, expires_at, created_at
		FROM refresh_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tokens []*models.RefreshToken
	for rows.Next() {
		token := &models.RefreshToken{}
		err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.TokenHash,
			&token.DeviceInfo,
			&token.ExpiresAt,
			&token.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	
	return tokens, rows.Err()
}

func (r *RefreshTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`
	
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	
	if result.RowsAffected() == 0 {
		return domerrors.ErrInvalidToken
	}
	
	return nil
}

func (r *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < $1`
	_, err := r.db.Exec(ctx, query, time.Now())
	return err
}
