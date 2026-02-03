package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	VPN      VPNConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type VPNConfig struct {
	WireGuardEnabled  bool
	WireGuardEndpoint string
	WireGuardPort     int
	WireGuardSubnet   string
	MaxPeersPerUser   int
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:  time.Duration(getEnvAsInt("SERVER_READ_TIMEOUT", 10)) * time.Second,
			WriteTimeout: time.Duration(getEnvAsInt("SERVER_WRITE_TIMEOUT", 10)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "vpn"),
			Password: getEnv("DB_PASSWORD", "vpn123"),
			DBName:   getEnv("DB_NAME", "vpn"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "your-access-secret-change-me"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "your-refresh-secret-change-me"),
			AccessTTL:     time.Duration(getEnvAsInt("JWT_ACCESS_TTL", 15)) * time.Minute,
			RefreshTTL:    time.Duration(getEnvAsInt("JWT_REFRESH_TTL", 168)) * time.Hour,
		},
		VPN: VPNConfig{
			WireGuardEnabled:  getEnvAsBool("WIREGUARD_ENABLED", true),
			WireGuardEndpoint: getEnv("WIREGUARD_ENDPOINT", ""),
			WireGuardPort:     getEnvAsInt("WIREGUARD_PORT", 51820),
			WireGuardSubnet:   getEnv("WIREGUARD_SUBNET", "10.13.13.0/24"),
			MaxPeersPerUser:   getEnvAsInt("MAX_PEERS_PER_USER", 5),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.JWT.AccessSecret == "" || c.JWT.AccessSecret == "your-access-secret-change-me" {
		return fmt.Errorf("JWT_ACCESS_SECRET must be set")
	}
	if c.JWT.RefreshSecret == "" || c.JWT.RefreshSecret == "your-refresh-secret-change-me" {
		return fmt.Errorf("JWT_REFRESH_SECRET must be set")
	}
	if c.VPN.WireGuardEnabled && c.VPN.WireGuardEndpoint == "" {
		return fmt.Errorf("WIREGUARD_ENDPOINT must be set when WireGuard is enabled")
	}
	return nil
}

func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}
