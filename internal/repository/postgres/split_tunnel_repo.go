package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/ObsidianServer/internal/domain/models"
)

type SplitTunnelRuleRepository struct {
	db *pgxpool.Pool
}

func NewSplitTunnelRuleRepository(db *pgxpool.Pool) *SplitTunnelRuleRepository {
	return &SplitTunnelRuleRepository{db: db}
}

func (r *SplitTunnelRuleRepository) Create(ctx context.Context, rule *models.SplitTunnelRule) error {
	query := `
		INSERT INTO split_tunnel_rules (
			id, peer_id, rule_type, value, resolved_ips, description,
			is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		rule.ID,
		rule.PeerID,
		rule.RuleType,
		rule.Value,
		rule.ResolvedIPs,
		rule.Description,
		rule.IsActive,
		rule.CreatedAt,
		rule.UpdatedAt,
	)
	return err
}

func (r *SplitTunnelRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SplitTunnelRule, error) {
	query := `
		SELECT id, peer_id, rule_type, value, resolved_ips, description,
			is_active, created_at, updated_at
		FROM split_tunnel_rules
		WHERE id = $1
	`

	rule := &models.SplitTunnelRule{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rule.ID,
		&rule.PeerID,
		&rule.RuleType,
		&rule.Value,
		&rule.ResolvedIPs,
		&rule.Description,
		&rule.IsActive,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("split tunnel rule not found")
		}
		return nil, err
	}

	return rule, nil
}

func (r *SplitTunnelRuleRepository) GetByPeerID(ctx context.Context, peerID uuid.UUID) ([]*models.SplitTunnelRule, error) {
	query := `
		SELECT id, peer_id, rule_type, value, resolved_ips, description,
			is_active, created_at, updated_at
		FROM split_tunnel_rules
		WHERE peer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, peerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.SplitTunnelRule
	for rows.Next() {
		rule := &models.SplitTunnelRule{}
		err := rows.Scan(
			&rule.ID,
			&rule.PeerID,
			&rule.RuleType,
			&rule.Value,
			&rule.ResolvedIPs,
			&rule.Description,
			&rule.IsActive,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

func (r *SplitTunnelRuleRepository) GetActiveByPeerID(ctx context.Context, peerID uuid.UUID) ([]*models.SplitTunnelRule, error) {
	query := `
		SELECT id, peer_id, rule_type, value, resolved_ips, description,
			is_active, created_at, updated_at
		FROM split_tunnel_rules
		WHERE peer_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, peerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.SplitTunnelRule
	for rows.Next() {
		rule := &models.SplitTunnelRule{}
		err := rows.Scan(
			&rule.ID,
			&rule.PeerID,
			&rule.RuleType,
			&rule.Value,
			&rule.ResolvedIPs,
			&rule.Description,
			&rule.IsActive,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

func (r *SplitTunnelRuleRepository) Update(ctx context.Context, rule *models.SplitTunnelRule) error {
	query := `
		UPDATE split_tunnel_rules
		SET rule_type = $2, value = $3, resolved_ips = $4, description = $5,
			is_active = $6, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		rule.ID,
		rule.RuleType,
		rule.Value,
		rule.ResolvedIPs,
		rule.Description,
		rule.IsActive,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("split tunnel rule not found")
	}

	return nil
}

func (r *SplitTunnelRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM split_tunnel_rules WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("split tunnel rule not found")
	}

	return nil
}

func (r *SplitTunnelRuleRepository) DeleteByPeerID(ctx context.Context, peerID uuid.UUID) error {
	query := `DELETE FROM split_tunnel_rules WHERE peer_id = $1`
	_, err := r.db.Exec(ctx, query, peerID)
	return err
}
