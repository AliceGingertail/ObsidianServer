# VPN Server

Собственный VPN-сервис с поддержкой WireGuard и планами на OpenVPN.

## Возможности

- Регистрация и авторизация пользователей
- JWT аутентификация (Access + Refresh токены)
- Управление VPN устройствами (peers)
- Поддержка WireGuard
- Автоматическое выделение IP-адресов
- Генерация клиентских конфигураций
- REST API
- PostgreSQL база данных
- Поддержка OpenVPN (пока в разработке)

## Архитектура

```
├── cmd/obsidian-server/          # Точка входа
├── internal/
│   ├── api/                 # HTTP handlers и роутер
│   ├── config/              # Конфигурация
│   ├── domain/              # Модели и ошибки
│   ├── repository/          # Работа с БД
│   ├── services/            # Бизнес-логика
│   └── vpn/                 # VPN провайдеры
├── pkg/                     # Переиспользуемые утилиты
└── deployments/             # Docker и скрипты
```

## Требования

- Go 1.21+
- PostgreSQL 15+
- WireGuard
- Docker и Docker Compose (опционально)

## Быстрый старт (Docker)

1. Перейдите в директорию Docker:
```bash
cd deployments/docker
```

2. Создайте файл `.env`:
```bash
cat > .env << EOF
JWT_ACCESS_SECRET=your-super-secret-access-key-change-me
JWT_REFRESH_SECRET=your-super-secret-refresh-key-change-me
WIREGUARD_ENDPOINT=your-server-ip:51820
EOF
```

3. Запустите сервисы:
```bash
docker compose up --build -d
```

4. Проверьте статус:
```bash
docker compose ps
docker compose logs obsidian-server
```

5. Откройте в браузере:
   - API: `http://localhost:8081/api/`
   - Админ-панель: `http://localhost:8081/admin/`

### Создание администратора

```bash
# Регистрация пользователя
curl -X POST http://localhost:8081/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@example.com","password":"password123"}'

# Назначение прав администратора
docker exec vpn-postgres psql -U vpn -d vpn -c \
  "UPDATE users SET is_admin=true WHERE email='admin@example.com';"
```

## Установка без Docker

### 1. Установите зависимости

**WireGuard:**
```bash
# Ubuntu/Debian
sudo apt install wireguard wireguard-tools

# Alpine
sudo apk add wireguard-tools
```

**PostgreSQL:**
```bash
sudo apt install postgresql postgresql-client
```

### 2. Настройте базу данных

```bash
sudo -u postgres psql
```

```sql
CREATE DATABASE vpn;
CREATE USER vpn WITH PASSWORD 'vpn123';
GRANT ALL PRIVILEGES ON DATABASE vpn TO vpn;
\q
```

Примените миграции:
```bash
psql -U vpn -d vpn -f internal/repository/migrations/001_init_schema.sql
psql -U vpn -d vpn -f internal/repository/migrations/002_add_username.sql
```

### 3. Настройте WireGuard

```bash
# Генерируем ключи
wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey

# Создаем конфигурацию
sudo nano /etc/wireguard/wg0.conf
```

Пример `wg0.conf`:
```ini
[Interface]
PrivateKey = <PRIVATE_KEY>
Address = 10.13.13.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
```

Запустите интерфейс:
```bash
sudo wg-quick up wg0
sudo systemctl enable wg-quick@wg0
```

### 4. Соберите и запустите сервер

```bash
# Скопируйте .env
cp .env.example .env
# Отредактируйте .env

# Соберите
go build -o obsidian-server ./cmd/obsidian-server

# Запустите
sudo ./obsidian-server
```

## API Документация

### Аутентификация

**Регистрация:**
```bash
curl -X POST http://localhost:8081/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john",
    "email": "john@example.com",
    "password": "password123"
  }'
```

**Вход:**
```bash
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "password123"
  }'
```

Ответ:
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "abc123...",
  "user": {
    "id": "uuid",
    "username": "john",
    "email": "john@example.com",
    "is_active": true,
    "is_admin": false
  }
}
```

### Управление устройствами (peers)

**Создать устройство:**
```bash
curl -X POST http://localhost:8081/api/vpn/peers \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "device_name": "Laptop",
    "protocol": "wireguard"
  }'
```

Ответ:
```json
{
  "peer": {
    "id": "uuid",
    "device_name": "Laptop",
    "protocol": "wireguard",
    "wg_ip_address": "10.13.13.2/24"
  },
  "config": "[Interface]\nPrivateKey = ...\n[Peer]\n..."
}
```

**Список устройств:**
```bash
curl http://localhost:8081/api/vpn/peers \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

**Получить конфигурацию:**
```bash
curl http://localhost:8081/api/vpn/peers/<PEER_ID>/config \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

**Удалить устройство:**
```bash
curl -X DELETE http://localhost:8081/api/vpn/peers/<PEER_ID> \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

## Настройка клиента

После создания peer скопируйте конфигурацию в файл `client.conf` и используйте:

**Linux/macOS:**
```bash
sudo wg-quick up ./client.conf
```

**Windows:**
1. Установите WireGuard GUI
2. Импортируйте конфигурацию
3. Активируйте туннель

## Структура базы данных

- `users` - пользователи (username, email, password, is_admin, is_active)
- `peers` - устройства пользователей (device_name, protocol, wg_public_key, wg_ip_address)
- `refresh_tokens` - refresh токены для сессий

## Переменные окружения

См. `.env.example` для полного списка.

Основные:
- `SERVER_PORT` - порт HTTP API (по умолчанию: 8080)
- `WIREGUARD_ENDPOINT` - публичный адрес сервера
- `WIREGUARD_SUBNET` - подсеть VPN (по умолчанию: 10.13.13.0/24)
- `MAX_PEERS_PER_USER` - максимум устройств на пользователя

## Админ-панель

Веб-интерфейс для управления сервером доступен по адресу: `http://localhost:8081/admin/`

### Возможности админ-панели

- Статистика: количество пользователей, пиров, активных подключений
- Управление пользователями: просмотр, удаление
- Управление пирами: просмотр, создание, удаление
- Настройки сервера: просмотр текущей конфигурации

### Доступ к админ-панели

1. Создайте пользователя через API или зарегистрируйтесь
2. Назначьте права администратора через базу данных:
```bash
# Подключение к PostgreSQL в Docker
docker exec -it vpn-postgres psql -U vpn -d vpn

# Назначить администратора
UPDATE users SET is_admin = true WHERE username = 'admin';
```

3. Откройте `http://localhost:8081/admin/` и войдите с учетными данными

### API админ-панели

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /api/admin/stats | Статистика сервера |
| GET | /api/admin/server-info | Информация о настройках |
| GET | /api/admin/users | Список пользователей |
| DELETE | /api/admin/users/{id} | Удалить пользователя |
| GET | /api/admin/peers | Список всех пиров |
| POST | /api/admin/peers | Создать пир для пользователя |
| DELETE | /api/admin/peers/{id} | Удалить пир |
