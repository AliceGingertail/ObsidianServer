#!/bin/bash
set -e

WG_CONFIG="/etc/wireguard/wg0.conf"
WG_INTERFACE="wg0"
WG_PORT="${WIREGUARD_PORT:-51820}"
WG_SUBNET="${WIREGUARD_SUBNET:-10.13.13.0/24}"

echo "Configuring WireGuard interface $WG_INTERFACE..."

# Проверяем, существует ли конфигурация
if [ ! -f "$WG_CONFIG" ]; then
    echo "Generating new WireGuard configuration..."
    
    # Генерируем ключи
    PRIVATE_KEY=$(wg genkey)
    PUBLIC_KEY=$(echo "$PRIVATE_KEY" | wg pubkey)
    
    # Создаем конфигурацию
    cat > "$WG_CONFIG" <<EOF
[Interface]
PrivateKey = $PRIVATE_KEY
Address = 10.13.13.1/24
ListenPort = $WG_PORT
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

# Peers will be added dynamically by the application
EOF
    
    chmod 600 "$WG_CONFIG"
    
    echo "WireGuard configuration created"
    echo "Server Public Key: $PUBLIC_KEY"
else
    echo "Using existing WireGuard configuration"
fi

# Запускаем WireGuard интерфейс
if ! ip link show "$WG_INTERFACE" &> /dev/null; then
    echo "Starting WireGuard interface..."
    wg-quick up "$WG_INTERFACE" || {
        echo "Warning: Failed to start WireGuard with wg-quick, trying manual setup..."
        
        # Manual setup fallback
        ip link add dev "$WG_INTERFACE" type wireguard
        wg setconf "$WG_INTERFACE" "$WG_CONFIG"
        ip addr add 10.13.13.1/24 dev "$WG_INTERFACE"
        ip link set up dev "$WG_INTERFACE"
    }
    
    echo "WireGuard interface started"
else
    echo "WireGuard interface already running"
fi

# Показываем статус
wg show "$WG_INTERFACE"
