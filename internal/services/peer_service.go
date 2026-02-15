package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourusername/ObsidianServer/internal/domain/models"
	domerrors "github.com/yourusername/ObsidianServer/internal/domain/errors"
	"github.com/yourusername/ObsidianServer/internal/repository"
	"github.com/yourusername/ObsidianServer/internal/repository/postgres"
	"github.com/yourusername/ObsidianServer/internal/vpn"
)

type PeerService struct {
	peerRepo        repository.PeerRepository
	vpnRegistry     *vpn.Registry
	maxPeersPerUser int
	subnet          string
}

func NewPeerService(
	peerRepo repository.PeerRepository,
	vpnRegistry *vpn.Registry,
	maxPeersPerUser int,
	subnet string,
) *PeerService {
	return &PeerService{
		peerRepo:        peerRepo,
		vpnRegistry:     vpnRegistry,
		maxPeersPerUser: maxPeersPerUser,
		subnet:          subnet,
	}
}

type CreatePeerRequest struct {
	UserID          uuid.UUID
	DeviceName      string
	Protocol        models.Protocol
	PublicKey       string // Публичный ключ, сгенерированный клиентом
	PrivateKey      string // Приватный ключ (только для server-generated peers)
	ServerGenerated bool   // Флаг: ключи сгенерированы сервером
}

type PeerWithConfig struct {
	Peer   *models.Peer `json:"peer"`
	Config string       `json:"config"`
}

func (s *PeerService) CreatePeer(ctx context.Context, req *CreatePeerRequest) (*PeerWithConfig, error) {
	// Проверяем лимит устройств
	count, err := s.peerRepo.CountByUserID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to count peers: %w", err)
	}

	if count >= s.maxPeersPerUser {
		return nil, domerrors.ErrMaxPeersReached
	}

	// Получаем VPN провайдер
	provider, err := s.vpnRegistry.Get(req.Protocol)
	if err != nil {
		return nil, err
	}

	// Выделяем IP-адрес
	ipAddress, err := s.allocateIPAddress(ctx, req.Protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}

	// Создаем peer
	peer := models.NewPeer(req.UserID, req.DeviceName, req.Protocol)

	// Заполняем данные в зависимости от протокола
	switch req.Protocol {
	case models.ProtocolWireGuard:
		peer.WGPublicKey = &req.PublicKey
		if req.PrivateKey != "" {
			peer.WGPrivateKey = &req.PrivateKey
		}
		peer.ServerGenerated = req.ServerGenerated
		// Генерируем preshared key на сервере для дополнительной защиты
		preshared, err := provider.GeneratePresharedKey(ctx)
		if err == nil && preshared != "" {
			peer.WGPreshared = &preshared
		}
		ipWithMask := postgres.FormatIPWithMask(ipAddress, s.subnet)
		peer.WGIPAddress = &ipWithMask
	case models.ProtocolOpenVPN:
		// Будет реализовано позже
		credentials, err := provider.GenerateCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate credentials: %w", err)
		}
		peer.OVPNCertificate = &credentials.Certificate
		peer.OVPNIPAddress = &ipAddress
	default:
		return nil, domerrors.ErrInvalidProtocol
	}

	// Сохраняем в БД
	if err := s.peerRepo.Create(ctx, peer); err != nil {
		return nil, fmt.Errorf("failed to save peer: %w", err)
	}

	// Добавляем peer в VPN конфигурацию
	if err := provider.AddPeer(ctx, peer); err != nil {
		// Откатываем создание peer в БД
		_ = s.peerRepo.Delete(ctx, peer.ID)
		return nil, fmt.Errorf("failed to add peer to VPN: %w", err)
	}

	// Генерируем частичную конфигурацию для клиента (без приватного ключа)
	config, err := provider.GenerateClientConfig(ctx, peer)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client config: %w", err)
	}

	return &PeerWithConfig{
		Peer:   peer,
		Config: config,
	}, nil
}

func (s *PeerService) GetPeersByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Peer, error) {
	return s.peerRepo.GetByUserID(ctx, userID)
}

func (s *PeerService) GetPeerByID(ctx context.Context, peerID, userID uuid.UUID) (*models.Peer, error) {
	peer, err := s.peerRepo.GetByID(ctx, peerID)
	if err != nil {
		return nil, err
	}

	// Проверяем, что peer принадлежит пользователю
	if peer.UserID != userID {
		return nil, domerrors.ErrForbidden
	}

	return peer, nil
}

func (s *PeerService) GetPeerConfig(ctx context.Context, peerID, userID uuid.UUID) (string, error) {
	peer, err := s.GetPeerByID(ctx, peerID, userID)
	if err != nil {
		return "", err
	}

	provider, err := s.vpnRegistry.Get(peer.Protocol)
	if err != nil {
		return "", err
	}

	return provider.GenerateClientConfig(ctx, peer)
}

// GetPeerConfigAdmin generates config without ownership check (admin only).
func (s *PeerService) GetPeerConfigAdmin(ctx context.Context, peerID uuid.UUID) (string, error) {
	peer, err := s.peerRepo.GetByID(ctx, peerID)
	if err != nil {
		return "", err
	}

	provider, err := s.vpnRegistry.Get(peer.Protocol)
	if err != nil {
		return "", err
	}

	return provider.GenerateClientConfig(ctx, peer)
}

func (s *PeerService) GetPeerConfigWithAllowedIPs(ctx context.Context, peerID, userID uuid.UUID, allowedIPs string) (string, error) {
	peer, err := s.GetPeerByID(ctx, peerID, userID)
	if err != nil {
		return "", err
	}

	provider, err := s.vpnRegistry.Get(peer.Protocol)
	if err != nil {
		return "", err
	}

	return provider.GenerateClientConfigWithAllowedIPs(ctx, peer, allowedIPs)
}

func (s *PeerService) DeletePeer(ctx context.Context, peerID, userID uuid.UUID) error {
	peer, err := s.GetPeerByID(ctx, peerID, userID)
	if err != nil {
		return err
	}

	// Удаляем из VPN конфигурации
	provider, err := s.vpnRegistry.Get(peer.Protocol)
	if err == nil {
		// Игнорируем ошибку удаления из VPN, т.к. peer может быть уже удален
		_ = provider.RemovePeer(ctx, peer)
	}

	// Удаляем из БД
	return s.peerRepo.Delete(ctx, peerID)
}

func (s *PeerService) UpdatePeer(ctx context.Context, peer *models.Peer) error {
	return s.peerRepo.Update(ctx, peer)
}

func (s *PeerService) allocateIPAddress(ctx context.Context, protocol models.Protocol) (string, error) {
	lastIP, err := s.peerRepo.GetLastIPAddress(ctx, protocol)
	if err != nil {
		return "", err
	}

	return postgres.AllocateNextIP(s.subnet, lastIP)
}

func (s *PeerService) GetAll(ctx context.Context) ([]*models.Peer, error) {
	return s.peerRepo.GetAll(ctx)
}

func (s *PeerService) Count(ctx context.Context) (int, error) {
	return s.peerRepo.Count(ctx)
}

func (s *PeerService) CountActive(ctx context.Context) (int, error) {
	return s.peerRepo.CountActive(ctx)
}

func (s *PeerService) DeletePeerByID(ctx context.Context, peerID uuid.UUID) error {
	peer, err := s.peerRepo.GetByID(ctx, peerID)
	if err != nil {
		return err
	}

	// Remove from VPN configuration
	provider, err := s.vpnRegistry.Get(peer.Protocol)
	if err == nil {
		_ = provider.RemovePeer(ctx, peer)
	}

	return s.peerRepo.Delete(ctx, peerID)
}
