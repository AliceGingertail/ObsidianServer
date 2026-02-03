package wireguard

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/yourusername/ObsidianServer/internal/domain/models"
	"github.com/yourusername/ObsidianServer/internal/vpn"
)

type Config struct {
	Endpoint    string // Публичный адрес сервера (например, vpn.example.com:51820)
	Subnet      string // Подсеть VPN (например, 10.13.13.0/24)
	Interface   string // Имя интерфейса (обычно wg0)
	DNS         string // DNS сервер для клиентов (опционально)
	AllowedIPs  string // Разрешенные IP для клиентов (обычно 0.0.0.0/0)
}

type Manager struct {
	config         *Config
	serverPublicKey string
}

func NewManager(config *Config) (*Manager, error) {
	if config.Interface == "" {
		config.Interface = "wg0"
	}
	if config.AllowedIPs == "" {
		config.AllowedIPs = "0.0.0.0/0, ::/0"
	}
	if config.DNS == "" {
		config.DNS = "1.1.1.1, 8.8.8.8"
	}

	m := &Manager{
		config: config,
	}

	// Получаем публичный ключ сервера
	pubKey, err := GetServerPublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get server public key: %w", err)
	}
	m.serverPublicKey = pubKey

	return m, nil
}

func (m *Manager) Name() models.Protocol {
	return models.ProtocolWireGuard
}

func (m *Manager) IsAvailable() bool {
	// Проверяем наличие команды wg
	cmd := exec.Command("wg", "show", m.config.Interface)
	return cmd.Run() == nil
}

func (m *Manager) GenerateCredentials(ctx context.Context) (*vpn.PeerCredentials, error) {
	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	preshared, err := GeneratePresharedKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate preshared key: %w", err)
	}

	return &vpn.PeerCredentials{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Preshared:  preshared,
	}, nil
}

func (m *Manager) AddPeer(ctx context.Context, peer *models.Peer) error {
	if peer.WGPublicKey == nil || peer.WGIPAddress == nil {
		return fmt.Errorf("peer missing WireGuard credentials")
	}

	// Формируем команду wg set
	args := []string{
		"set", m.config.Interface,
		"peer", *peer.WGPublicKey,
		"allowed-ips", *peer.WGIPAddress,
	}

	if peer.WGPreshared != nil && *peer.WGPreshared != "" {
		args = append(args, "preshared-key", "/dev/stdin")
	}

	cmd := exec.CommandContext(ctx, "wg", args...)
	
	if peer.WGPreshared != nil && *peer.WGPreshared != "" {
		cmd.Stdin = strings.NewReader(*peer.WGPreshared)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add peer to WireGuard: %w (output: %s)", err, string(output))
	}

	// Сохраняем конфигурацию
	if err := m.saveConfig(ctx); err != nil {
		return fmt.Errorf("failed to save WireGuard config: %w", err)
	}

	return nil
}

func (m *Manager) RemovePeer(ctx context.Context, peer *models.Peer) error {
	if peer.WGPublicKey == nil {
		return fmt.Errorf("peer missing public key")
	}

	cmd := exec.CommandContext(ctx, "wg", "set", m.config.Interface,
		"peer", *peer.WGPublicKey,
		"remove")

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove peer from WireGuard: %w (output: %s)", err, string(output))
	}

	// Сохраняем конфигурацию
	if err := m.saveConfig(ctx); err != nil {
		return fmt.Errorf("failed to save WireGuard config: %w", err)
	}

	return nil
}

func (m *Manager) GenerateClientConfig(ctx context.Context, peer *models.Peer) (string, error) {
	if peer.WGPrivateKey == nil || peer.WGIPAddress == nil {
		return "", fmt.Errorf("peer missing WireGuard credentials")
	}

	var config strings.Builder
	
	config.WriteString("[Interface]\n")
	config.WriteString(fmt.Sprintf("PrivateKey = %s\n", *peer.WGPrivateKey))
	config.WriteString(fmt.Sprintf("Address = %s\n", *peer.WGIPAddress))
	if m.config.DNS != "" {
		config.WriteString(fmt.Sprintf("DNS = %s\n", m.config.DNS))
	}
	config.WriteString("\n")
	
	config.WriteString("[Peer]\n")
	config.WriteString(fmt.Sprintf("PublicKey = %s\n", m.serverPublicKey))
	config.WriteString(fmt.Sprintf("Endpoint = %s\n", m.config.Endpoint))
	config.WriteString(fmt.Sprintf("AllowedIPs = %s\n", m.config.AllowedIPs))
	
	if peer.WGPreshared != nil && *peer.WGPreshared != "" {
		config.WriteString(fmt.Sprintf("PresharedKey = %s\n", *peer.WGPreshared))
	}
	
	config.WriteString("PersistentKeepalive = 25\n")

	return config.String(), nil
}

func (m *Manager) GetServerInfo(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"public_key": m.serverPublicKey,
		"endpoint":   m.config.Endpoint,
		"subnet":     m.config.Subnet,
		"dns":        m.config.DNS,
		"interface":  m.config.Interface,
	}, nil
}

func (m *Manager) saveConfig(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "wg-quick", "save", m.config.Interface)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("wg-quick save failed: %w (output: %s)", err, string(output))
	}
	return nil
}
