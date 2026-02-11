package vpn

import (
	"context"

	"github.com/yourusername/ObsidianServer/internal/domain/models"
)

// Provider определяет интерфейс для VPN провайдеров
type Provider interface {
	// Name возвращает название протокола
	Name() models.Protocol

	// GenerateCredentials генерирует ключи и конфигурацию для нового peer
	GenerateCredentials(ctx context.Context) (*PeerCredentials, error)

	// GeneratePresharedKey генерирует только preshared key
	GeneratePresharedKey(ctx context.Context) (string, error)

	// AddPeer добавляет peer в конфигурацию VPN сервера
	AddPeer(ctx context.Context, peer *models.Peer) error

	// RemovePeer удаляет peer из конфигурации VPN сервера
	RemovePeer(ctx context.Context, peer *models.Peer) error

	// GenerateClientConfig генерирует конфигурационный файл для клиента
	GenerateClientConfig(ctx context.Context, peer *models.Peer) (string, error)

	// GenerateClientConfigWithAllowedIPs generates config with custom AllowedIPs for split tunneling
	GenerateClientConfigWithAllowedIPs(ctx context.Context, peer *models.Peer, allowedIPs string) (string, error)

	// GetServerInfo возвращает информацию о сервере (публичный ключ, endpoint, etc)
	GetServerInfo(ctx context.Context) (map[string]string, error)

	// IsAvailable проверяет доступность провайдера
	IsAvailable() bool
}

// PeerCredentials содержит сгенерированные учетные данные
type PeerCredentials struct {
	PublicKey  string
	PrivateKey string
	Preshared  string
	// Для OpenVPN
	Certificate string
}
