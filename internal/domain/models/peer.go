package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Peer struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	DeviceName string    `json:"device_name"`
	Protocol   Protocol  `json:"protocol"`

	// WireGuard specific fields
	WGPublicKey  *string `json:"wg_public_key,omitempty"`
	WGPrivateKey *string `json:"wg_private_key,omitempty"`
	WGPreshared  *string `json:"wg_preshared,omitempty"`
	WGIPAddress  *string `json:"wg_ip_address,omitempty"`

	// OpenVPN specific fields (future)
	OVPNCertificate *string `json:"ovpn_certificate,omitempty"`
	OVPNPrivateKey  *string `json:"ovpn_private_key,omitempty"`
	OVPNIPAddress   *string `json:"ovpn_ip_address,omitempty"`

	// Split tunneling
	SplitTunnelMode SplitTunnelMode `json:"split_tunnel_mode"`

	IsActive  bool       `json:"is_active"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func NewPeer(userID uuid.UUID, deviceName string, protocol Protocol) *Peer {
	now := time.Now()
	return &Peer{
		ID:              uuid.New(),
		UserID:          userID,
		DeviceName:      deviceName,
		Protocol:        protocol,
		SplitTunnelMode: SplitTunnelModeAll,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// GetIPWithoutMask returns the IP address without CIDR mask
func (p *Peer) GetIPWithoutMask() string {
	if p.WGIPAddress == nil {
		return ""
	}
	ip := *p.WGIPAddress
	if idx := strings.Index(ip, "/"); idx != -1 {
		return ip[:idx]
	}
	return ip
}

// Config представляет конфигурацию для клиента
type PeerConfig struct {
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
	DNS        string `json:"dns,omitempty"`

	// Server info
	PublicKey  string `json:"public_key"`
	Endpoint   string `json:"endpoint"`
	AllowedIPs string `json:"allowed_ips"`
	Preshared  string `json:"preshared,omitempty"`
}
