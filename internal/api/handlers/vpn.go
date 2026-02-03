package handlers

import (
	"net/http"

	"github.com/yourusername/ObsidianServer/internal/services"
)

type VPNHandler struct {
	vpnService *services.VPNService
}

func NewVPNHandler(vpnService *services.VPNService) *VPNHandler {
	return &VPNHandler{
		vpnService: vpnService,
	}
}

// GetServerInfo возвращает информацию о VPN сервере
// GET /api/vpn/server-info
func (h *VPNHandler) GetServerInfo(w http.ResponseWriter, r *http.Request) {
	serverInfo, err := h.vpnService.GetServerInfo(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get server info")
		return
	}

	respondJSON(w, http.StatusOK, serverInfo)
}
