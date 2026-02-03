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

type PeerHandler struct {
	peerService *services.PeerService
}

func NewPeerHandler(peerService *services.PeerService) *PeerHandler {
	return &PeerHandler{
		peerService: peerService,
	}
}

type createPeerRequest struct {
	DeviceName string          `json:"device_name"`
	Protocol   models.Protocol `json:"protocol"`
}

func (h *PeerHandler) CreatePeer(w http.ResponseWriter, r *http.Request) {
	var req createPeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DeviceName == "" {
		respondError(w, http.StatusBadRequest, "Device name is required")
		return
	}

	if !req.Protocol.IsValid() {
		respondError(w, http.StatusBadRequest, "Invalid protocol")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	peerWithConfig, err := h.peerService.CreatePeer(r.Context(), &services.CreatePeerRequest{
		UserID:     userID,
		DeviceName: req.DeviceName,
		Protocol:   req.Protocol,
	})

	if err != nil {
		if err == domerrors.ErrMaxPeersReached {
			respondError(w, http.StatusForbidden, "Maximum number of peers reached")
			return
		}
		if err == domerrors.ErrProviderNotFound {
			respondError(w, http.StatusBadRequest, "VPN protocol not available")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to create peer")
		return
	}

	respondJSON(w, http.StatusCreated, peerWithConfig)
}

func (h *PeerHandler) GetPeers(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())

	peers, err := h.peerService.GetPeersByUserID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get peers")
		return
	}

	if peers == nil {
		peers = []*models.Peer{}
	}

	respondJSON(w, http.StatusOK, peers)
}

func (h *PeerHandler) GetPeer(w http.ResponseWriter, r *http.Request) {
	peerIDStr := chi.URLParam(r, "id")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	userID := GetUserIDFromContext(r.Context())

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

	respondJSON(w, http.StatusOK, peer)
}

func (h *PeerHandler) GetPeerConfig(w http.ResponseWriter, r *http.Request) {
	peerIDStr := chi.URLParam(r, "id")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	config, err := h.peerService.GetPeerConfig(r.Context(), peerID, userID)
	if err != nil {
		if err == domerrors.ErrPeerNotFound {
			respondError(w, http.StatusNotFound, "Peer not found")
			return
		}
		if err == domerrors.ErrForbidden {
			respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get peer config")
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(config))
}

func (h *PeerHandler) DeletePeer(w http.ResponseWriter, r *http.Request) {
	peerIDStr := chi.URLParam(r, "id")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid peer ID")
		return
	}

	userID := GetUserIDFromContext(r.Context())

	if err := h.peerService.DeletePeer(r.Context(), peerID, userID); err != nil {
		if err == domerrors.ErrPeerNotFound {
			respondError(w, http.StatusNotFound, "Peer not found")
			return
		}
		if err == domerrors.ErrForbidden {
			respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to delete peer")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Peer deleted successfully"})
}
