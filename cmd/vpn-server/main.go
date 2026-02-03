package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/ObsidianServer/internal/api"
	"github.com/yourusername/ObsidianServer/internal/config"
	"github.com/yourusername/ObsidianServer/internal/repository/postgres"
	"github.com/yourusername/ObsidianServer/internal/services"
	"github.com/yourusername/ObsidianServer/internal/vpn"
	"github.com/yourusername/ObsidianServer/internal/vpn/wireguard"
	"github.com/yourusername/ObsidianServer/pkg/jwt"
)

func main() {
	fmt.Println("Hello World")

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Starting VPN Server...")

	// Подключаемся к БД
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	log.Println("Connected to database")

	// Проверяем подключение
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Инициализируем репозитории
	userRepo := postgres.NewUserRepository(dbPool)
	peerRepo := postgres.NewPeerRepository(dbPool)
	tokenRepo := postgres.NewRefreshTokenRepository(dbPool)

	// Инициализируем JWT manager
	jwtManager := jwt.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	// Инициализируем VPN registry
	vpnRegistry := vpn.NewRegistry()

	// Регистрируем WireGuard провайдер
	if cfg.VPN.WireGuardEnabled {
		wgManager, err := wireguard.NewManager(&wireguard.Config{
			Endpoint:   cfg.VPN.WireGuardEndpoint,
			Subnet:     cfg.VPN.WireGuardSubnet,
			Interface:  "wg0",
			DNS:        "1.1.1.1, 8.8.8.8",
			AllowedIPs: "0.0.0.0/0, ::/0",
		})

		if err != nil {
			log.Printf("Warning: Failed to initialize WireGuard: %v", err)
		} else {
			if err := vpnRegistry.Register(wgManager); err != nil {
				log.Fatalf("Failed to register WireGuard provider: %v", err)
			}
			log.Println("WireGuard provider registered")
		}
	}

	// Инициализируем сервисы
	authService := services.NewAuthService(userRepo, tokenRepo, jwtManager)
	userService := services.NewUserService(userRepo)
	peerService := services.NewPeerService(
		peerRepo,
		vpnRegistry,
		cfg.VPN.MaxPeersPerUser,
		cfg.VPN.WireGuardSubnet,
	)
	vpnService := services.NewVPNService(vpnRegistry)

	// Создаем роутер с конфигурацией сервера
	serverCfg := &api.ServerConfig{
		WireGuardEnabled:  cfg.VPN.WireGuardEnabled,
		WireGuardEndpoint: cfg.VPN.WireGuardEndpoint,
		WireGuardPort:     cfg.VPN.WireGuardPort,
		WireGuardSubnet:   cfg.VPN.WireGuardSubnet,
		MaxPeersPerUser:   cfg.VPN.MaxPeersPerUser,
	}
	router := api.NewRouterWithConfig(authService, userService, peerService, vpnService, jwtManager, serverCfg)
	handler := router.Setup()

	// Настраиваем HTTP сервер
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
