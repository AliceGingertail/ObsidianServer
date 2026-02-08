#!/bin/bash
set -e

echo "Starting VPN Server entrypoint..."

# Автогенерация JWT секретов если не заданы
SECRETS_FILE="/etc/wireguard/.jwt_secrets"

if [ -z "$JWT_ACCESS_SECRET" ] || [ -z "$JWT_REFRESH_SECRET" ]; then
    echo "JWT secrets not provided, checking for saved secrets..."

    if [ -f "$SECRETS_FILE" ]; then
        echo "Loading saved JWT secrets..."
        source "$SECRETS_FILE"
    else
        echo "Generating new JWT secrets..."
        JWT_ACCESS_SECRET=$(openssl rand -base64 32)
        JWT_REFRESH_SECRET=$(openssl rand -base64 32)

        # Сохраняем для переиспользования после рестарта
        cat > "$SECRETS_FILE" << EOF
export JWT_ACCESS_SECRET="$JWT_ACCESS_SECRET"
export JWT_REFRESH_SECRET="$JWT_REFRESH_SECRET"
EOF
        chmod 600 "$SECRETS_FILE"
        echo "JWT secrets generated and saved"
    fi

    export JWT_ACCESS_SECRET
    export JWT_REFRESH_SECRET
fi

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
