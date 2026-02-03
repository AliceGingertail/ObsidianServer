package wireguard

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/crypto/curve25519"
)

// GeneratePrivateKey генерирует WireGuard приватный ключ
func GeneratePrivateKey() (string, error) {
	// Генерируем 32 случайных байта
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// WireGuard требует специальной обработки ключа
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64

	return base64.StdEncoding.EncodeToString(key), nil
}

// GeneratePublicKey вычисляет публичный ключ из приватного
func GeneratePublicKey(privateKey string) (string, error) {
	// Декодируем приватный ключ
	privKeyBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privKeyBytes) != 32 {
		return "", fmt.Errorf("invalid private key length: %d", len(privKeyBytes))
	}

	// Вычисляем публичный ключ используя curve25519
	var privateKeyArray [32]byte
	copy(privateKeyArray[:], privKeyBytes)

	publicKeyArray, err := curve25519.X25519(privateKeyArray[:], curve25519.Basepoint)
	if err != nil {
		return "", fmt.Errorf("failed to generate public key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(publicKeyArray), nil
}

// GeneratePresharedKey генерирует preshared ключ
func GeneratePresharedKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate preshared key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

// GenerateKeyPair генерирует пару приватный+публичный ключ
func GenerateKeyPair() (privateKey, publicKey string, err error) {
	privateKey, err = GeneratePrivateKey()
	if err != nil {
		return "", "", err
	}

	publicKey, err = GeneratePublicKey(privateKey)
	if err != nil {
		return "", "", err
	}

	return privateKey, publicKey, nil
}

// GetServerPublicKey получает публичный ключ сервера из wg0
func GetServerPublicKey() (string, error) {
	cmd := exec.Command("wg", "show", "wg0", "public-key")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get server public key: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}
