package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/yourusername/ObsidianServer/internal/api/handlers"
	"github.com/yourusername/ObsidianServer/internal/api/middleware"
	"github.com/yourusername/ObsidianServer/internal/services"
	"github.com/yourusername/ObsidianServer/pkg/jwt"
)

type Router struct {
	authHandler        *handlers.AuthHandler
	userHandler        *handlers.UserHandler
	peerHandler        *handlers.PeerHandler
	vpnHandler         *handlers.VPNHandler
	adminHandler       *handlers.AdminHandler
	splitTunnelHandler *handlers.SplitTunnelHandler
	authMW             *middleware.AuthMiddleware
}

type ServerConfig struct {
	WireGuardEnabled  bool
	WireGuardEndpoint string
	WireGuardPort     int
	WireGuardSubnet   string
	MaxPeersPerUser   int
}

func NewRouter(
	authService *services.AuthService,
	userService *services.UserService,
	peerService *services.PeerService,
	vpnService *services.VPNService,
	splitTunnelService *services.SplitTunnelService,
	jwtManager *jwt.Manager,
) *Router {
	return &Router{
		authHandler:        handlers.NewAuthHandler(authService),
		userHandler:        handlers.NewUserHandler(userService),
		peerHandler:        handlers.NewPeerHandlerWithSplitTunnel(peerService, splitTunnelService),
		vpnHandler:         handlers.NewVPNHandler(vpnService),
		adminHandler:       handlers.NewAdminHandler(userService, peerService),
		splitTunnelHandler: handlers.NewSplitTunnelHandler(splitTunnelService, peerService),
		authMW:             middleware.NewAuthMiddleware(jwtManager),
	}
}

func NewRouterWithConfig(
	authService *services.AuthService,
	userService *services.UserService,
	peerService *services.PeerService,
	vpnService *services.VPNService,
	splitTunnelService *services.SplitTunnelService,
	jwtManager *jwt.Manager,
	serverCfg *ServerConfig,
) *Router {
	serverInfo := &handlers.ServerInfoResponse{
		WireGuardEnabled:  serverCfg.WireGuardEnabled,
		WireGuardEndpoint: serverCfg.WireGuardEndpoint,
		WireGuardPort:     serverCfg.WireGuardPort,
		WireGuardSubnet:   serverCfg.WireGuardSubnet,
		MaxPeersPerUser:   serverCfg.MaxPeersPerUser,
		ServerVersion:     "1.0.0",
	}

	return &Router{
		authHandler:        handlers.NewAuthHandler(authService),
		userHandler:        handlers.NewUserHandler(userService),
		peerHandler:        handlers.NewPeerHandlerWithSplitTunnel(peerService, splitTunnelService),
		vpnHandler:         handlers.NewVPNHandler(vpnService),
		adminHandler:       handlers.NewAdminHandlerWithConfig(userService, peerService, serverInfo),
		splitTunnelHandler: handlers.NewSplitTunnelHandler(splitTunnelService, peerService),
		authMW:             middleware.NewAuthMiddleware(jwtManager),
	}
}

func (rt *Router) Setup() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Public auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", rt.authHandler.Register)
			r.Post("/login", rt.authHandler.Login)
			r.Post("/refresh", rt.authHandler.Refresh)

			// Protected auth routes
			r.Group(func(r chi.Router) {
				r.Use(rt.authMW.Authenticate)
				r.Post("/logout", rt.authHandler.Logout)
				r.Post("/logout-all", rt.authHandler.LogoutAll)
			})
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(rt.authMW.Authenticate)

			// User routes
			r.Route("/user", func(r chi.Router) {
				r.Get("/me", rt.userHandler.GetMe)
			})

			// VPN/Peer routes
			r.Route("/vpn", func(r chi.Router) {
				// Server info
				r.Get("/server-info", rt.vpnHandler.GetServerInfo)

				// Peers management
				r.Route("/peers", func(r chi.Router) {
					r.Get("/", rt.peerHandler.GetPeers)
					r.Post("/", rt.peerHandler.CreatePeer)
					r.Get("/{id}", rt.peerHandler.GetPeer)
					r.Get("/{id}/config", rt.peerHandler.GetPeerConfig)
					r.Delete("/{id}", rt.peerHandler.DeletePeer)

					// Split tunnel routes
					r.Route("/{peerID}/split-tunnel", func(r chi.Router) {
						r.Put("/mode", rt.splitTunnelHandler.SetSplitTunnelMode)
						r.Get("/", rt.splitTunnelHandler.GetRules)
						r.Post("/rules", rt.splitTunnelHandler.CreateRule)
						r.Delete("/rules/{ruleID}", rt.splitTunnelHandler.DeleteRule)
						r.Post("/refresh-domains", rt.splitTunnelHandler.RefreshDomains)
						r.Get("/config", rt.splitTunnelHandler.GetConfigWithSplitTunnel)
					})
				})
			})

			// Admin routes (requires admin access)
			r.Route("/admin", func(r chi.Router) {
				r.Use(rt.authMW.RequireAdmin)
				r.Get("/stats", rt.adminHandler.GetStats)
				r.Get("/server-info", rt.adminHandler.GetServerInfo)
				r.Get("/users", rt.adminHandler.GetAllUsers)
				r.Delete("/users/{id}", rt.adminHandler.DeleteUser)
				r.Get("/peers", rt.adminHandler.GetAllPeers)
				r.Post("/peers", rt.adminHandler.CreatePeer)
				r.Delete("/peers/{id}", rt.adminHandler.DeletePeer)
			})
		})
	})

	// Admin panel static files
	fileServer := http.FileServer(http.Dir("web/admin"))
	r.Handle("/admin", http.RedirectHandler("/admin/", http.StatusMovedPermanently))
	r.Handle("/admin/*", http.StripPrefix("/admin", fileServerWithIndex(fileServer)))

	return r
}

// fileServerWithIndex wraps a file server to serve index.html for directory requests
func fileServerWithIndex(fs http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If path ends with / or has no extension, serve index.html
		if strings.HasSuffix(r.URL.Path, "/") || !strings.Contains(r.URL.Path, ".") {
			r.URL.Path = "/"
		}
		fs.ServeHTTP(w, r)
	}
}
