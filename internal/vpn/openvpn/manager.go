package openvpn

import (
	"context"
	"fmt"

	"github.com/yourusername/ObsidianServer/internal/domain/models"
	"github.com/yourusername/ObsidianServer/internal/vpn"
)

// Manager реализует VPN провайдер для OpenVPN
// TODO: Implement OpenVPN support in Phase 3
type Manager struct {
	// Configuration will be added later
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Name() models.Protocol {
	return models.ProtocolOpenVPN
}

func (m *Manager) IsAvailable() bool {
	// OpenVPN не реализован пока
	return false
}

func (m *Manager) GenerateCredentials(ctx context.Context) (*vpn.PeerCredentials, error) {
	return nil, fmt.Errorf("OpenVPN support not implemented yet")
}

func (m *Manager) AddPeer(ctx context.Context, peer *models.Peer) error {
	return fmt.Errorf("OpenVPN support not implemented yet")
}

func (m *Manager) RemovePeer(ctx context.Context, peer *models.Peer) error {
	return fmt.Errorf("OpenVPN support not implemented yet")
}

func (m *Manager) GenerateClientConfig(ctx context.Context, peer *models.Peer) (string, error) {
	return "", fmt.Errorf("OpenVPN support not implemented yet")
}

func (m *Manager) GetServerInfo(ctx context.Context) (map[string]string, error) {
	return nil, fmt.Errorf("OpenVPN support not implemented yet")
}
