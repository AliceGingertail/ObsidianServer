package services

import (
	"context"

	"github.com/yourusername/ObsidianServer/internal/domain/models"
	"github.com/yourusername/ObsidianServer/internal/vpn"
)

type VPNService struct {
	vpnRegistry *vpn.Registry
}

func NewVPNService(vpnRegistry *vpn.Registry) *VPNService {
	return &VPNService{
		vpnRegistry: vpnRegistry,
	}
}

type ServerInfo struct {
	AvailableProtocols []ProtocolInfo `json:"available_protocols"`
}

type ProtocolInfo struct {
	Protocol   models.Protocol            `json:"protocol"`
	Available  bool                       `json:"available"`
	ServerInfo map[string]string          `json:"server_info,omitempty"`
}

// GetServerInfo возвращает информацию о доступных VPN протоколах и серверах
func (s *VPNService) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	protocolsInfo := make([]ProtocolInfo, 0)
	
	// WireGuard
	wgProvider, err := s.vpnRegistry.Get(models.ProtocolWireGuard)
	if err == nil && wgProvider.IsAvailable() {
		info, err := wgProvider.GetServerInfo(ctx)
		if err == nil {
			protocolsInfo = append(protocolsInfo, ProtocolInfo{
				Protocol:   models.ProtocolWireGuard,
				Available:  true,
				ServerInfo: info,
			})
		}
	} else {
		protocolsInfo = append(protocolsInfo, ProtocolInfo{
			Protocol:  models.ProtocolWireGuard,
			Available: false,
		})
	}
	
	// OpenVPN (в будущем)
	ovpnProvider, err := s.vpnRegistry.Get(models.ProtocolOpenVPN)
	if err == nil && ovpnProvider.IsAvailable() {
		info, err := ovpnProvider.GetServerInfo(ctx)
		if err == nil {
			protocolsInfo = append(protocolsInfo, ProtocolInfo{
				Protocol:   models.ProtocolOpenVPN,
				Available:  true,
				ServerInfo: info,
			})
		}
	} else {
		protocolsInfo = append(protocolsInfo, ProtocolInfo{
			Protocol:  models.ProtocolOpenVPN,
			Available: false,
		})
	}
	
	return &ServerInfo{
		AvailableProtocols: protocolsInfo,
	}, nil
}
