package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yourusername/ObsidianServer/internal/domain/models"
	"github.com/yourusername/ObsidianServer/internal/services"
)

type AdminHandler struct {
	userService *services.UserService
	peerService *services.PeerService
	serverInfo  *ServerInfoResponse
}

type ServerInfoResponse struct {
	WireGuardEnabled  bool   `json:"wireguard_enabled"`
	WireGuardEndpoint string `json:"wireguard_endpoint"`
	WireGuardPort     int    `json:"wireguard_port"`
	WireGuardSubnet   string `json:"wireguard_subnet"`
	MaxPeersPerUser   int    `json:"max_peers_per_user"`
	ServerVersion     string `json:"server_version"`
}

func NewAdminHandler(userService *services.UserService, peerService *services.PeerService) *AdminHandler {
	return &AdminHandler{
		userService: userService,
		peerService: peerService,
	}
}

func NewAdminHandlerWithConfig(userService *services.UserService, peerService *services.PeerService, serverInfo *ServerInfoResponse) *AdminHandler {
	return &AdminHandler{
		userService: userService,
		peerService: peerService,
		serverInfo:  serverInfo,
	}
}

type StatsResponse struct {
	TotalUsers  int `json:"total_users"`
	TotalPeers  int `json:"total_peers"`
	ActivePeers int `json:"active_peers"`
}

func (h *AdminHandler) GetServerInfo(w http.ResponseWriter, r *http.Request) {
	if h.serverInfo == nil {
		respondError(w, http.StatusInternalServerError, "Server info not configured")
		return
	}
	respondJSON(w, http.StatusOK, h.serverInfo)
}

func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	userCount, err := h.userService.Count(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to count users")
		return
	}

	peerCount, err := h.peerService.Count(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to count peers")
		return
	}

	activePeers, err := h.peerService.CountActive(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to count active peers")
		return
	}

	respondJSON(w, http.StatusOK, StatsResponse{
		TotalUsers:  userCount,
		TotalPeers:  peerCount,
		ActivePeers: activePeers,
	})
}

func (h *AdminHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userService.GetAll(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get users")
		return
	}

	respondJSON(w, http.StatusOK, users)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := h.userService.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) GetAllPeers(w http.ResponseWriter, r *http.Request) {
	peers, err := h.peerService.GetAll(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get peers")
		return
	}

	respondJSON(w, http.StatusOK, peers)
}

func (h *AdminHandler) DeletePeer(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	if err := h.peerService.DeletePeerByID(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete peer")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type adminCreatePeerRequest struct {
	UserID     string          `json:"user_id"`
	DeviceName string          `json:"device_name"`
	Protocol   models.Protocol `json:"protocol"`
}

type updateEndpointRequest struct {
	Endpoint string `json:"endpoint"`
}

const serverConfigPath = "/etc/wireguard/.server_config"

// UpdateEndpoint updates the WireGuard endpoint
func (h *AdminHandler) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	var req updateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	endpoint := strings.TrimSpace(req.Endpoint)
	if endpoint == "" {
		respondError(w, http.StatusBadRequest, "Endpoint is required")
		return
	}

	// Валидация формата endpoint (ip:port или domain:port)
	endpointRegex := regexp.MustCompile(`^[\w\.\-]+:\d+$`)
	if !endpointRegex.MatchString(endpoint) {
		respondError(w, http.StatusBadRequest, "Invalid endpoint format. Use: ip:port or domain:port")
		return
	}

	// Сохраняем в файл конфигурации
	content := fmt.Sprintf("export WIREGUARD_ENDPOINT=\"%s\"\n", endpoint)
	if err := os.WriteFile(serverConfigPath, []byte(content), 0600); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save endpoint")
		return
	}

	// Обновляем переменную окружения для текущего процесса
	os.Setenv("WIREGUARD_ENDPOINT", endpoint)

	// Обновляем serverInfo если он есть
	if h.serverInfo != nil {
		h.serverInfo.WireGuardEndpoint = endpoint
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message":  "Endpoint updated successfully",
		"endpoint": endpoint,
	})
}

func (h *AdminHandler) CreatePeer(w http.ResponseWriter, r *http.Request) {
	var req adminCreatePeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if req.DeviceName == "" {
		respondError(w, http.StatusBadRequest, "Device name is required")
		return
	}

	if !req.Protocol.IsValid() {
		respondError(w, http.StatusBadRequest, "Invalid protocol (use 'wireguard' or 'openvpn')")
		return
	}

	peerWithConfig, err := h.peerService.CreatePeer(r.Context(), &services.CreatePeerRequest{
		UserID:     userID,
		DeviceName: req.DeviceName,
		Protocol:   req.Protocol,
	})

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create peer: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, peerWithConfig)
}
