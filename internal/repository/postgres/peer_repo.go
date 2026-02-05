package postgres

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/ObsidianServer/internal/domain/models"
	domerrors "github.com/yourusername/ObsidianServer/internal/domain/errors"
)

type PeerRepository struct {
	db *pgxpool.Pool
}

func NewPeerRepository(db *pgxpool.Pool) *PeerRepository {
	return &PeerRepository{db: db}
}

func (r *PeerRepository) Create(ctx context.Context, peer *models.Peer) error {
	query := `
		INSERT INTO peers (
			id, user_id, device_name, protocol,
			wg_public_key, wg_private_key, wg_preshared, wg_ip_address,
			ovpn_certificate, ovpn_private_key, ovpn_ip_address,
			split_tunnel_mode,
			is_active, last_seen, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	_, err := r.db.Exec(ctx, query,
		peer.ID,
		peer.UserID,
		peer.DeviceName,
		peer.Protocol,
		peer.WGPublicKey,
		peer.WGPrivateKey,
		peer.WGPreshared,
		peer.WGIPAddress,
		peer.OVPNCertificate,
		peer.OVPNPrivateKey,
		peer.OVPNIPAddress,
		peer.SplitTunnelMode,
		peer.IsActive,
		peer.LastSeen,
		peer.CreatedAt,
		peer.UpdatedAt,
	)
	return err
}

func (r *PeerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Peer, error) {
	query := `
		SELECT id, user_id, device_name, protocol,
			wg_public_key, wg_private_key, wg_preshared, wg_ip_address,
			ovpn_certificate, ovpn_private_key, ovpn_ip_address,
			split_tunnel_mode,
			is_active, last_seen, created_at, updated_at
		FROM peers
		WHERE id = $1
	`

	peer := &models.Peer{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&peer.ID,
		&peer.UserID,
		&peer.DeviceName,
		&peer.Protocol,
		&peer.WGPublicKey,
		&peer.WGPrivateKey,
		&peer.WGPreshared,
		&peer.WGIPAddress,
		&peer.OVPNCertificate,
		&peer.OVPNPrivateKey,
		&peer.OVPNIPAddress,
		&peer.SplitTunnelMode,
		&peer.IsActive,
		&peer.LastSeen,
		&peer.CreatedAt,
		&peer.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domerrors.ErrPeerNotFound
		}
		return nil, err
	}

	return peer, nil
}

func (r *PeerRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Peer, error) {
	query := `
		SELECT id, user_id, device_name, protocol,
			wg_public_key, wg_private_key, wg_preshared, wg_ip_address,
			ovpn_certificate, ovpn_private_key, ovpn_ip_address,
			split_tunnel_mode,
			is_active, last_seen, created_at, updated_at
		FROM peers
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*models.Peer
	for rows.Next() {
		peer := &models.Peer{}
		err := rows.Scan(
			&peer.ID,
			&peer.UserID,
			&peer.DeviceName,
			&peer.Protocol,
			&peer.WGPublicKey,
			&peer.WGPrivateKey,
			&peer.WGPreshared,
			&peer.WGIPAddress,
			&peer.OVPNCertificate,
			&peer.OVPNPrivateKey,
			&peer.OVPNIPAddress,
			&peer.SplitTunnelMode,
			&peer.IsActive,
			&peer.LastSeen,
			&peer.CreatedAt,
			&peer.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}

	return peers, rows.Err()
}

func (r *PeerRepository) GetByPublicKey(ctx context.Context, publicKey string) (*models.Peer, error) {
	query := `
		SELECT id, user_id, device_name, protocol,
			wg_public_key, wg_private_key, wg_preshared, wg_ip_address,
			ovpn_certificate, ovpn_private_key, ovpn_ip_address,
			split_tunnel_mode,
			is_active, last_seen, created_at, updated_at
		FROM peers
		WHERE wg_public_key = $1
	`

	peer := &models.Peer{}
	err := r.db.QueryRow(ctx, query, publicKey).Scan(
		&peer.ID,
		&peer.UserID,
		&peer.DeviceName,
		&peer.Protocol,
		&peer.WGPublicKey,
		&peer.WGPrivateKey,
		&peer.WGPreshared,
		&peer.WGIPAddress,
		&peer.OVPNCertificate,
		&peer.OVPNPrivateKey,
		&peer.OVPNIPAddress,
		&peer.SplitTunnelMode,
		&peer.IsActive,
		&peer.LastSeen,
		&peer.CreatedAt,
		&peer.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domerrors.ErrPeerNotFound
		}
		return nil, err
	}

	return peer, nil
}

func (r *PeerRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM peers WHERE user_id = $1`
	
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *PeerRepository) Update(ctx context.Context, peer *models.Peer) error {
	query := `
		UPDATE peers
		SET device_name = $2, protocol = $3,
			wg_public_key = $4, wg_private_key = $5, wg_preshared = $6, wg_ip_address = $7,
			ovpn_certificate = $8, ovpn_private_key = $9, ovpn_ip_address = $10,
			split_tunnel_mode = $11,
			is_active = $12, last_seen = $13
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		peer.ID,
		peer.DeviceName,
		peer.Protocol,
		peer.WGPublicKey,
		peer.WGPrivateKey,
		peer.WGPreshared,
		peer.WGIPAddress,
		peer.OVPNCertificate,
		peer.OVPNPrivateKey,
		peer.OVPNIPAddress,
		peer.SplitTunnelMode,
		peer.IsActive,
		peer.LastSeen,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domerrors.ErrPeerNotFound
	}

	return nil
}

func (r *PeerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM peers WHERE id = $1`
	
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	
	if result.RowsAffected() == 0 {
		return domerrors.ErrPeerNotFound
	}
	
	return nil
}

func (r *PeerRepository) GetLastIPAddress(ctx context.Context, protocol models.Protocol) (string, error) {
	var query string
	switch protocol {
	case models.ProtocolWireGuard:
		query = `
			SELECT wg_ip_address FROM peers
			WHERE protocol = 'wireguard' AND wg_ip_address IS NOT NULL
			ORDER BY wg_ip_address DESC
			LIMIT 1
		`
	case models.ProtocolOpenVPN:
		query = `
			SELECT ovpn_ip_address FROM peers
			WHERE protocol = 'openvpn' AND ovpn_ip_address IS NOT NULL
			ORDER BY ovpn_ip_address DESC
			LIMIT 1
		`
	default:
		return "", domerrors.ErrInvalidProtocol
	}
	
	var ipAddress *string
	err := r.db.QueryRow(ctx, query).Scan(&ipAddress)
	
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // No peers yet
		}
		return "", err
	}
	
	if ipAddress == nil {
		return "", nil
	}
	
	return *ipAddress, nil
}

func (r *PeerRepository) GetAll(ctx context.Context) ([]*models.Peer, error) {
	query := `
		SELECT id, user_id, device_name, protocol,
			wg_public_key, wg_private_key, wg_preshared, wg_ip_address,
			ovpn_certificate, ovpn_private_key, ovpn_ip_address,
			split_tunnel_mode,
			is_active, last_seen, created_at, updated_at
		FROM peers
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*models.Peer
	for rows.Next() {
		peer := &models.Peer{}
		err := rows.Scan(
			&peer.ID,
			&peer.UserID,
			&peer.DeviceName,
			&peer.Protocol,
			&peer.WGPublicKey,
			&peer.WGPrivateKey,
			&peer.WGPreshared,
			&peer.WGIPAddress,
			&peer.OVPNCertificate,
			&peer.OVPNPrivateKey,
			&peer.OVPNIPAddress,
			&peer.SplitTunnelMode,
			&peer.IsActive,
			&peer.LastSeen,
			&peer.CreatedAt,
			&peer.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}

	return peers, rows.Err()
}

func (r *PeerRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM peers`

	var count int
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

func (r *PeerRepository) CountActive(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM peers WHERE is_active = true`

	var count int
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// AllocateIP выделяет следующий доступный IP-адрес из подсети
func AllocateNextIP(subnet, lastIP string) (string, error) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", err
	}
	
	var nextIP net.IP
	if lastIP == "" {
		// Первый IP: берем базовый адрес подсети + 2 (например, 10.13.13.2)
		nextIP = make(net.IP, len(ipNet.IP))
		copy(nextIP, ipNet.IP)
		nextIP[len(nextIP)-1] += 2
	} else {
		// Парсим последний IP и увеличиваем
		lastIPParsed := net.ParseIP(lastIP)
		if lastIPParsed == nil {
			return "", fmt.Errorf("invalid last IP: %s", lastIP)
		}
		
		nextIP = make(net.IP, len(lastIPParsed))
		copy(nextIP, lastIPParsed)
		
		// Увеличиваем последний октет
		nextIP[len(nextIP)-1]++
	}
	
	// Проверяем, что IP в пределах подсети
	if !ipNet.Contains(nextIP) {
		return "", fmt.Errorf("subnet exhausted")
	}
	
	return nextIP.String(), nil
}

// FormatIPWithMask добавляет маску к IP-адресу
func FormatIPWithMask(ip, subnet string) string {
	parts := strings.Split(subnet, "/")
	if len(parts) != 2 {
		return ip
	}
	return ip + "/" + parts[1]
}
