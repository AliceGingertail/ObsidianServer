package vpn

import (
	"fmt"
	"sync"

	"github.com/yourusername/ObsidianServer/internal/domain/models"
	domerrors "github.com/yourusername/ObsidianServer/internal/domain/errors"
)

// Registry управляет доступными VPN провайдерами
type Registry struct {
	providers map[models.Protocol]Provider
	mu        sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[models.Protocol]Provider),
	}
}

// Register регистрирует новый VPN провайдер
func (r *Registry) Register(provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	protocol := provider.Name()
	if _, exists := r.providers[protocol]; exists {
		return fmt.Errorf("provider for protocol %s already registered", protocol)
	}

	r.providers[protocol] = provider
	return nil
}

// Get возвращает провайдер для указанного протокола
func (r *Registry) Get(protocol models.Protocol) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[protocol]
	if !exists {
		return nil, domerrors.ErrProviderNotFound
	}

	if !provider.IsAvailable() {
		return nil, fmt.Errorf("provider %s is not available", protocol)
	}

	return provider, nil
}

// List возвращает список доступных протоколов
func (r *Registry) List() []models.Protocol {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protocols := make([]models.Protocol, 0, len(r.providers))
	for protocol, provider := range r.providers {
		if provider.IsAvailable() {
			protocols = append(protocols, protocol)
		}
	}

	return protocols
}
