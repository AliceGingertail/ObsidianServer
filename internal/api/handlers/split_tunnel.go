package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yourusername/ObsidianServer/internal/domain/models"
	domerrors "github.com/yourusername/ObsidianServer/internal/domain/errors"
	"github.com/yourusername/ObsidianServer/internal/services"
)

type SplitTunnelHandler struct {
	splitTunnelService *services.SplitTunnelService
	peerService        *services.PeerService
}

func NewSplitTunnelHandler(
	splitTunnelService *services.SplitTunnelService,
	peerService *services.PeerService,
) *SplitTunnelHandler {
	return &SplitTunnelHandler{
		splitTunnelService: splitTunnelService,
		peerService:        peerService,
	}
}

type setSplitTunnelModeRequest struct {
	Mode models.SplitTunnelMode `json:"mode"`
}

type createRuleRequest struct {
	RuleType    models.RuleType `json:"rule_type"`
	Value       string          `json:"value"`
	Description string          `json:"description,omitempty"`
}

// SetSplitTunnelMode sets the split tunnel mode for a peer
func (h *SplitTunnelHandler) SetSplitTunnelMode(w http.ResponseWriter, r *http.Request) {
	peerIDStr := chi.URLParam(r, "peerID")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	// Verify peer ownership
	_, err = h.peerService.GetPeerByID(r.Context(), peerID, userID)
	if err != nil {
		if err == domerrors.ErrPeerNotFound {
			respondError(w, http.StatusNotFound, "Peer not found")
			return
		}
		if err == domerrors.ErrForbidden {
			respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to verify peer ownership")
		return
	}

	var req setSplitTunnelModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !req.Mode.IsValid() {
		respondError(w, http.StatusBadRequest, "Invalid split tunnel mode. Use: all, include, exclude")
		return
	}

	if err := h.splitTunnelService.SetPeerSplitTunnelMode(r.Context(), peerID, req.Mode); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to set split tunnel mode")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Split tunnel mode updated",
		"mode":    string(req.Mode),
	})
}

// GetRules returns all split tunnel rules for a peer
func (h *SplitTunnelHandler) GetRules(w http.ResponseWriter, r *http.Request) {
	peerIDStr := chi.URLParam(r, "peerID")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	// Verify peer ownership
	peer, err := h.peerService.GetPeerByID(r.Context(), peerID, userID)
	if err != nil {
		if err == domerrors.ErrPeerNotFound {
			respondError(w, http.StatusNotFound, "Peer not found")
			return
		}
		if err == domerrors.ErrForbidden {
			respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to verify peer ownership")
		return
	}

	rules, err := h.splitTunnelService.GetRulesByPeerID(r.Context(), peerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get rules")
		return
	}

	if rules == nil {
		rules = []*models.SplitTunnelRule{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"mode":  peer.SplitTunnelMode,
		"rules": rules,
	})
}

// CreateRule creates a new split tunnel rule for a peer
func (h *SplitTunnelHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	peerIDStr := chi.URLParam(r, "peerID")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	// Verify peer ownership
	_, err = h.peerService.GetPeerByID(r.Context(), peerID, userID)
	if err != nil {
		if err == domerrors.ErrPeerNotFound {
			respondError(w, http.StatusNotFound, "Peer not found")
			return
		}
		if err == domerrors.ErrForbidden {
			respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to verify peer ownership")
		return
	}

	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !req.RuleType.IsValid() {
		respondError(w, http.StatusBadRequest, "Invalid rule type. Use: ip, cidr, domain")
		return
	}

	if req.Value == "" {
		respondError(w, http.StatusBadRequest, "Value is required")
		return
	}

	rule, err := h.splitTunnelService.CreateRule(r.Context(), &services.CreateRuleRequest{
		PeerID:      peerID,
		RuleType:    req.RuleType,
		Value:       req.Value,
		Description: req.Description,
	})

	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, rule)
}

// DeleteRule deletes a split tunnel rule
func (h *SplitTunnelHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	peerIDStr := chi.URLParam(r, "peerID")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	// Verify peer ownership
	_, err = h.peerService.GetPeerByID(r.Context(), peerID, userID)
	if err != nil {
		if err == domerrors.ErrPeerNotFound {
			respondError(w, http.StatusNotFound, "Peer not found")
			return
		}
		if err == domerrors.ErrForbidden {
			respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to verify peer ownership")
		return
	}

	if err := h.splitTunnelService.DeleteRule(r.Context(), ruleID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete rule")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Rule deleted"})
}

// RefreshDomains refreshes DNS resolution for domain rules
func (h *SplitTunnelHandler) RefreshDomains(w http.ResponseWriter, r *http.Request) {
	peerIDStr := chi.URLParam(r, "peerID")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	// Verify peer ownership
	_, err = h.peerService.GetPeerByID(r.Context(), peerID, userID)
	if err != nil {
		if err == domerrors.ErrPeerNotFound {
			respondError(w, http.StatusNotFound, "Peer not found")
			return
		}
		if err == domerrors.ErrForbidden {
			respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to verify peer ownership")
		return
	}

	if err := h.splitTunnelService.RefreshDomainResolutions(r.Context(), peerID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to refresh domains")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Domains refreshed"})
}

// GetConfigWithSplitTunnel returns the peer config with split tunnel rules applied
func (h *SplitTunnelHandler) GetConfigWithSplitTunnel(w http.ResponseWriter, r *http.Request) {
	peerIDStr := chi.URLParam(r, "peerID")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	// Get peer with ownership check
	peer, err := h.peerService.GetPeerByID(r.Context(), peerID, userID)
	if err != nil {
		if err == domerrors.ErrPeerNotFound {
			respondError(w, http.StatusNotFound, "Peer not found")
			return
		}
		if err == domerrors.ErrForbidden {
			respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get peer")
		return
	}

	// Get allowed IPs based on split tunnel configuration
	allowedIPs, err := h.splitTunnelService.GetAllowedIPs(r.Context(), peer)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to calculate allowed IPs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"peer_id":     peer.ID,
		"mode":        peer.SplitTunnelMode,
		"allowed_ips": allowedIPs,
	})
}
