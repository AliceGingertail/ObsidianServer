#!/bin/bash
set -e

echo "Starting VPN Server entrypoint..."

# Загрузка модуля WireGuard
if ! lsmod | grep -q wireguard; then
    echo "Loading WireGuard kernel module..."
    modprobe wireguard || echo "Warning: Failed to load WireGuard module"
fi

# Настройка WireGuard
if [ "$WIREGUARD_ENABLED" = "true" ]; then
    echo "Setting up WireGuard..."
    /app/setup-wg.sh
fi

# Настройка iptables для NAT
echo "Configuring iptables..."
iptables -t nat -A POSTROUTING -s ${WIREGUARD_SUBNET:-10.13.13.0/24} -o eth0 -j MASQUERADE
iptables -A FORWARD -i wg0 -j ACCEPT
iptables -A FORWARD -o wg0 -j ACCEPT

echo "Entrypoint completed, starting application..."

# Запуск приложения
exec "$@"
