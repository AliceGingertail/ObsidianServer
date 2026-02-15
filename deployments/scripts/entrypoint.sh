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

# Загрузка/установка WIREGUARD_ENDPOINT
if [ -n "$WIREGUARD_ENDPOINT" ]; then
    echo "Using provided endpoint: $WIREGUARD_ENDPOINT"
else
    # Определяем IP напрямую с физического интерфейса.
    # curl ipify.org ненадёжен: если на хосте есть VPN (wg0), трафик пойдёт
    # через него и вернёт чужой exit IP.
    #
    # На VPS — это публичный IP (eth0/ens3).
    # Локально — LAN IP (wlan0), что корректно для тестирования.
    # Для VPS за cloud NAT — задайте WIREGUARD_ENDPOINT вручную.
    DEFAULT_ROUTE_IF=$(ip -4 route show default | grep -oP 'dev \K\S+' | head -1)
    PUBLIC_IP=$(ip -4 addr show "$DEFAULT_ROUTE_IF" 2>/dev/null | grep -oP 'inet \K[^/]+' | head -1)

    WG_PORT="${WIREGUARD_PORT:-51821}"
    if [ -n "$PUBLIC_IP" ] && [ "$PUBLIC_IP" != "127.0.0.1" ]; then
        WIREGUARD_ENDPOINT="${PUBLIC_IP}:${WG_PORT}"
        echo "Auto-detected endpoint: $WIREGUARD_ENDPOINT (from $DEFAULT_ROUTE_IF)"
    else
        WIREGUARD_ENDPOINT="localhost:${WG_PORT}"
        echo "ERROR: Could not detect IP from default interface!"
        echo "Set WIREGUARD_ENDPOINT in .env file"
    fi
fi

export WIREGUARD_ENDPOINT

# Включаем IP forwarding (с host network sysctls не работают в compose)
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.ipv6.conf.all.forwarding=1

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

# Определяем сетевой интерфейс для NAT (не hardcode eth0)
DEFAULT_IF=$(ip -4 route show default | grep -oP 'dev \K\S+' | head -1)
if [ -z "$DEFAULT_IF" ]; then
    DEFAULT_IF="eth0"
    echo "Warning: Could not detect default interface, using eth0"
fi
echo "Using network interface: $DEFAULT_IF"

WG_SUBNET="${WIREGUARD_SUBNET:-10.13.13.0/24}"
WG_IF="${WIREGUARD_INTERFACE:-wg1}"

# ─── Policy routing ────────────────────────────────────────────────────────
# Если на хосте есть другой VPN (wg0 с AllowedIPs=0.0.0.0/0), его маршруты
# перехватывают ВЕСЬ трафик, включая пересылаемый с wg1. Без policy routing
# пакеты клиентов уходят в чужой VPN вместо физического интерфейса.
#
# Решение: отдельная таблица маршрутов с физическим default route.
# Пакеты, приходящие на wg1, маркируются fwmark и маршрутизируются через неё.
# ───────────────────────────────────────────────────────────────────────────
WG_TABLE=51821
DEFAULT_GW=$(ip -4 route show default | awk '{print $3}' | head -1)

echo "Setting up policy routing for VPN forwarding..."
echo "  default gateway: $DEFAULT_GW via $DEFAULT_IF (table $WG_TABLE)"

# Таблица маршрутов: физический default route (в обход любых VPN на хосте)
ip route replace default via "$DEFAULT_GW" dev "$DEFAULT_IF" table $WG_TABLE 2>/dev/null || true

# Маркируем входящий трафик на VPN интерфейсе
iptables -t mangle -C PREROUTING -i "$WG_IF" -j MARK --set-mark $WG_TABLE 2>/dev/null \
    || iptables -t mangle -A PREROUTING -i "$WG_IF" -j MARK --set-mark $WG_TABLE

# Маршрутизируем маркированный трафик через нашу таблицу
ip rule add fwmark $WG_TABLE lookup $WG_TABLE priority 100 2>/dev/null || true

# ─── iptables для NAT (идемпотентно) ──────────────────────────────────────
echo "Configuring iptables..."

# NAT: MASQUERADE для VPN-трафика, выходящего в интернет
iptables -t nat -C POSTROUTING -s "$WG_SUBNET" -o "$DEFAULT_IF" -j MASQUERADE 2>/dev/null \
    || iptables -t nat -I POSTROUTING -s "$WG_SUBNET" -o "$DEFAULT_IF" -j MASQUERADE

# FORWARD: правила в основной цепочке
iptables -C FORWARD -i "$WG_IF" -j ACCEPT 2>/dev/null \
    || iptables -I FORWARD -i "$WG_IF" -j ACCEPT
iptables -C FORWARD -o "$WG_IF" -j ACCEPT 2>/dev/null \
    || iptables -I FORWARD -o "$WG_IF" -j ACCEPT

# DOCKER-USER: Docker не трогает эту цепочку — правила переживут перезапуск контейнеров.
# Без этого Docker может перезаписать FORWARD chain и заблокировать VPN-трафик.
if iptables -L DOCKER-USER -n &>/dev/null; then
    echo "Adding rules to DOCKER-USER chain..."
    iptables -C DOCKER-USER -i "$WG_IF" -j ACCEPT 2>/dev/null \
        || iptables -I DOCKER-USER -i "$WG_IF" -j ACCEPT
    iptables -C DOCKER-USER -o "$WG_IF" -j ACCEPT 2>/dev/null \
        || iptables -I DOCKER-USER -o "$WG_IF" -j ACCEPT
fi

# MSS clamping: предотвращает проблемы с большими TCP-пакетами через VPN.
# WireGuard добавляет ~80 байт overhead, без этого TCP-сессии могут зависать
# (handshake работает, но веб-страницы не грузятся).
iptables -t mangle -C FORWARD -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null \
    || iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu

echo "Entrypoint completed, starting application..."

# Запуск приложения
exec "$@"
