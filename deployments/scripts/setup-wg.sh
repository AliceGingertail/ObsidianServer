#!/bin/bash
set -e

WG_INTERFACE="${WIREGUARD_INTERFACE:-wg1}"
WG_CONFIG="/etc/wireguard/${WG_INTERFACE}.conf"
WG_PORT="${WIREGUARD_PORT:-51821}"
WG_SUBNET="${WIREGUARD_SUBNET:-10.13.13.0/24}"
WG_SERVER_IP=$(echo "$WG_SUBNET" | awk -F'[./]' '{printf "%d.%d.%d.1/%s", $1, $2, $3, $4}')

echo "Configuring WireGuard interface $WG_INTERFACE..."

# Проверяем, существует ли конфигурация
if [ ! -f "$WG_CONFIG" ]; then
    echo "Generating new WireGuard configuration..."

    # Генерируем ключи
    PRIVATE_KEY=$(wg genkey)
    PUBLIC_KEY=$(echo "$PRIVATE_KEY" | wg pubkey)

    # Создаем конфигурацию (wg-quick формат)
    # iptables/NAT настраиваются в entrypoint.sh, не в PostUp/PostDown,
    # чтобы избежать дублирования правил и привязки к имени интерфейса
    cat > "$WG_CONFIG" <<EOF
[Interface]
PrivateKey = $PRIVATE_KEY
Address = $WG_SERVER_IP
ListenPort = $WG_PORT
MTU = 1420

# Peers will be added dynamically by the application
EOF

    chmod 600 "$WG_CONFIG"

    echo "WireGuard configuration created"
    echo "Server Public Key: $PUBLIC_KEY"
else
    echo "Using existing WireGuard configuration"
fi

# network_mode: host — интерфейс живёт на хосте и переживает рестарт контейнера.
# Всегда удаляем и пересоздаём, чтобы гарантировать корректный IP-адрес, MTU
# и конфигурацию пиров из сохранённого конфига.
if ip link show "$WG_INTERFACE" &> /dev/null; then
    echo "Removing existing $WG_INTERFACE interface (will recreate)..."
    # wg-quick down корректно удаляет интерфейс и маршруты
    wg-quick down "$WG_INTERFACE" 2>/dev/null || ip link delete dev "$WG_INTERFACE" 2>/dev/null || true
fi

echo "Starting WireGuard interface..."
wg-quick up "$WG_INTERFACE" || {
    echo "Warning: Failed to start WireGuard with wg-quick, trying manual setup..."

    # wg setconf не понимает wg-quick директивы (Address, MTU, DNS, etc.)
    # Создаём чистый конфиг только с PrivateKey/ListenPort/Peer секциями
    WG_RAW_CONFIG=$(mktemp)
    grep -v -E '^\s*(Address|MTU|DNS|Table|PreUp|PostUp|PreDown|PostDown|SaveConfig)\s*=' "$WG_CONFIG" \
        | grep -v '^\s*#' | grep -v '^\s*$' > "$WG_RAW_CONFIG"
    echo "" >> "$WG_RAW_CONFIG"

    ip link add dev "$WG_INTERFACE" type wireguard
    wg setconf "$WG_INTERFACE" "$WG_RAW_CONFIG"
    ip addr add "$WG_SERVER_IP" dev "$WG_INTERFACE"
    ip link set mtu 1420 dev "$WG_INTERFACE"
    ip link set up dev "$WG_INTERFACE"

    rm -f "$WG_RAW_CONFIG"
}

echo "WireGuard interface started"

# Показываем статус
wg show "$WG_INTERFACE"
