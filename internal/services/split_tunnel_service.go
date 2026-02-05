package services

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/yourusername/ObsidianServer/internal/domain/models"
	"github.com/yourusername/ObsidianServer/internal/repository"
)

type SplitTunnelService struct {
	ruleRepo repository.SplitTunnelRuleRepository
	peerRepo repository.PeerRepository
}

func NewSplitTunnelService(
	ruleRepo repository.SplitTunnelRuleRepository,
	peerRepo repository.PeerRepository,
) *SplitTunnelService {
	return &SplitTunnelService{
		ruleRepo: ruleRepo,
		peerRepo: peerRepo,
	}
}

type CreateRuleRequest struct {
	PeerID      uuid.UUID
	RuleType    models.RuleType
	Value       string
	Description string
}

func (s *SplitTunnelService) CreateRule(ctx context.Context, req *CreateRuleRequest) (*models.SplitTunnelRule, error) {
	// Validate rule type
	if !req.RuleType.IsValid() {
		return nil, fmt.Errorf("invalid rule type: %s", req.RuleType)
	}

	// Validate value based on rule type
	if err := s.validateRuleValue(req.RuleType, req.Value); err != nil {
		return nil, err
	}

	rule := models.NewSplitTunnelRule(req.PeerID, req.RuleType, req.Value, req.Description)

	// If domain, try to resolve IPs
	if req.RuleType == models.RuleTypeDomain {
		ips, err := s.resolveDomain(req.Value)
		if err == nil && len(ips) > 0 {
			rule.ResolvedIPs = ips
		}
	}

	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	return rule, nil
}

func (s *SplitTunnelService) GetRulesByPeerID(ctx context.Context, peerID uuid.UUID) ([]*models.SplitTunnelRule, error) {
	return s.ruleRepo.GetByPeerID(ctx, peerID)
}

func (s *SplitTunnelService) GetActiveRulesByPeerID(ctx context.Context, peerID uuid.UUID) ([]*models.SplitTunnelRule, error) {
	return s.ruleRepo.GetActiveByPeerID(ctx, peerID)
}

func (s *SplitTunnelService) UpdateRule(ctx context.Context, rule *models.SplitTunnelRule) error {
	// Validate if changed
	if !rule.RuleType.IsValid() {
		return fmt.Errorf("invalid rule type: %s", rule.RuleType)
	}

	if err := s.validateRuleValue(rule.RuleType, rule.Value); err != nil {
		return err
	}

	// Re-resolve domain if needed
	if rule.RuleType == models.RuleTypeDomain {
		ips, err := s.resolveDomain(rule.Value)
		if err == nil && len(ips) > 0 {
			rule.ResolvedIPs = ips
		}
	}

	return s.ruleRepo.Update(ctx, rule)
}

func (s *SplitTunnelService) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	return s.ruleRepo.Delete(ctx, ruleID)
}

func (s *SplitTunnelService) SetPeerSplitTunnelMode(ctx context.Context, peerID uuid.UUID, mode models.SplitTunnelMode) error {
	if !mode.IsValid() {
		return fmt.Errorf("invalid split tunnel mode: %s", mode)
	}

	peer, err := s.peerRepo.GetByID(ctx, peerID)
	if err != nil {
		return err
	}

	peer.SplitTunnelMode = mode
	return s.peerRepo.Update(ctx, peer)
}

// GetAllowedIPs generates AllowedIPs string for WireGuard config based on split tunnel mode and rules
func (s *SplitTunnelService) GetAllowedIPs(ctx context.Context, peer *models.Peer) (string, error) {
	switch peer.SplitTunnelMode {
	case models.SplitTunnelModeAll:
		// All traffic through VPN
		return "0.0.0.0/0, ::/0", nil

	case models.SplitTunnelModeInclude:
		// Only listed routes through VPN
		rules, err := s.ruleRepo.GetActiveByPeerID(ctx, peer.ID)
		if err != nil {
			return "", err
		}

		if len(rules) == 0 {
			// No rules, just VPN subnet
			return peer.GetIPWithoutMask() + "/32", nil
		}

		var allowedIPs []string
		for _, rule := range rules {
			ips := s.ruleToIPs(rule)
			allowedIPs = append(allowedIPs, ips...)
		}

		// Always include VPN server IP for connectivity
		if peer.WGIPAddress != nil {
			// Add server subnet for internal VPN communication
			allowedIPs = append(allowedIPs, peer.GetIPWithoutMask()+"/32")
		}

		return strings.Join(allowedIPs, ", "), nil

	case models.SplitTunnelModeExclude:
		// All traffic except listed goes through VPN
		// This is complex in WireGuard - we return 0.0.0.0/0 and note that
		// client needs to set up local routes for excluded destinations
		// For true split tunneling exclusion, client-side routing is needed
		return "0.0.0.0/0, ::/0", nil

	default:
		return "0.0.0.0/0, ::/0", nil
	}
}

func (s *SplitTunnelService) validateRuleValue(ruleType models.RuleType, value string) error {
	switch ruleType {
	case models.RuleTypeIP:
		if ip := net.ParseIP(value); ip == nil {
			return fmt.Errorf("invalid IP address: %s", value)
		}
	case models.RuleTypeCIDR:
		if _, _, err := net.ParseCIDR(value); err != nil {
			return fmt.Errorf("invalid CIDR: %s", value)
		}
	case models.RuleTypeDomain:
		if value == "" {
			return fmt.Errorf("domain cannot be empty")
		}
		// Basic domain validation
		if strings.ContainsAny(value, " \t\n") {
			return fmt.Errorf("invalid domain: %s", value)
		}
	}
	return nil
}

func (s *SplitTunnelService) resolveDomain(domain string) ([]string, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, ip := range ips {
		result = append(result, ip.String())
	}
	return result, nil
}

func (s *SplitTunnelService) ruleToIPs(rule *models.SplitTunnelRule) []string {
	switch rule.RuleType {
	case models.RuleTypeIP:
		return []string{rule.Value + "/32"}
	case models.RuleTypeCIDR:
		return []string{rule.Value}
	case models.RuleTypeDomain:
		// Use resolved IPs if available
		if len(rule.ResolvedIPs) > 0 {
			var ips []string
			for _, ip := range rule.ResolvedIPs {
				ips = append(ips, ip+"/32")
			}
			return ips
		}
		// Try to resolve now
		resolved, err := s.resolveDomain(rule.Value)
		if err != nil {
			return nil
		}
		var ips []string
		for _, ip := range resolved {
			ips = append(ips, ip+"/32")
		}
		return ips
	}
	return nil
}

// RefreshDomainResolutions updates resolved IPs for all domain rules
func (s *SplitTunnelService) RefreshDomainResolutions(ctx context.Context, peerID uuid.UUID) error {
	rules, err := s.ruleRepo.GetByPeerID(ctx, peerID)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if rule.RuleType == models.RuleTypeDomain {
			ips, err := s.resolveDomain(rule.Value)
			if err == nil && len(ips) > 0 {
				rule.ResolvedIPs = ips
				if err := s.ruleRepo.Update(ctx, rule); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
