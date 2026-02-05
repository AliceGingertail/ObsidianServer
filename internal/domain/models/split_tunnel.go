package models

import (
	"time"

	"github.com/google/uuid"
)

type SplitTunnelMode string

const (
	SplitTunnelModeAll     SplitTunnelMode = "all"     // All traffic through VPN
	SplitTunnelModeInclude SplitTunnelMode = "include" // Only listed routes through VPN
	SplitTunnelModeExclude SplitTunnelMode = "exclude" // All except listed through VPN
)

func (m SplitTunnelMode) IsValid() bool {
	switch m {
	case SplitTunnelModeAll, SplitTunnelModeInclude, SplitTunnelModeExclude:
		return true
	}
	return false
}

type RuleType string

const (
	RuleTypeIP     RuleType = "ip"
	RuleTypeCIDR   RuleType = "cidr"
	RuleTypeDomain RuleType = "domain"
)

func (r RuleType) IsValid() bool {
	switch r {
	case RuleTypeIP, RuleTypeCIDR, RuleTypeDomain:
		return true
	}
	return false
}

type SplitTunnelRule struct {
	ID          uuid.UUID `json:"id"`
	PeerID      uuid.UUID `json:"peer_id"`
	RuleType    RuleType  `json:"rule_type"`
	Value       string    `json:"value"`
	ResolvedIPs []string  `json:"resolved_ips,omitempty"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewSplitTunnelRule(peerID uuid.UUID, ruleType RuleType, value, description string) *SplitTunnelRule {
	now := time.Now()
	return &SplitTunnelRule{
		ID:          uuid.New(),
		PeerID:      peerID,
		RuleType:    ruleType,
		Value:       value,
		Description: description,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
