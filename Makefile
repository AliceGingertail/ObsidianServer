.PHONY: help build run test clean docker-build docker-up docker-down fmt lint dev db-setup db-reset

# Переменные
BINARY_NAME=obsidian-server
MAIN_PATH=./cmd/vpn-server
DB_USER=obsidian
DB_PASSWORD=obsidian_dev
DB_NAME=obsidian_dev

help: ## Показать справку
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Разработка
build: ## Собрать бинарник
	go build -o $(BINARY_NAME) $(MAIN_PATH)

run: ## Запустить локально (нужен sudo для WireGuard)
	sudo go run $(MAIN_PATH)

dev: ## Запустить с hot-reload (требуется air)
	sudo air

test: ## Запустить тесты
	go test ./... -v

test-coverage: ## Тесты с покрытием
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

# Код
fmt: ## Форматирование кода
	go fmt ./...

lint: ## Линтер
	go vet ./...

tidy: ## Обновить зависимости
	go mod download
	go mod tidy

# База данных
db-setup: ## Создать БД и применить миграции
	@echo "Создание базы данных..."
	sudo -u postgres psql -c "CREATE USER $(DB_USER) WITH PASSWORD '$(DB_PASSWORD)';" || true
	sudo -u postgres psql -c "CREATE DATABASE $(DB_NAME) OWNER $(DB_USER);" || true
	@echo "Применение миграций..."
	PGPASSWORD=$(DB_PASSWORD) psql -U $(DB_USER) -d $(DB_NAME) -h localhost -f internal/repository/migrations/001_init_schema.sql

db-reset: ## Очистить данные
	PGPASSWORD=$(DB_PASSWORD) psql -U $(DB_USER) -d $(DB_NAME) -h localhost -c "TRUNCATE users, peers, refresh_tokens CASCADE;"

# Docker PostgreSQL (рекомендуется для разработки)
db-docker-start: ## Запустить PostgreSQL в Docker
	@echo "Запуск PostgreSQL в Docker..."
	@docker run -d --name obsidian-db \
		-e POSTGRES_USER=$(DB_USER) \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-e POSTGRES_DB=$(DB_NAME) \
		-p 5432:5432 \
		-v obsidian-db-data:/var/lib/postgresql/data \
		postgres:15-alpine 2>/dev/null || echo "Контейнер уже существует, используем существующий"
	@echo "Ждём пока PostgreSQL запустится..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if docker exec obsidian-db pg_isready -U $(DB_USER) > /dev/null 2>&1; then \
			echo "✓ PostgreSQL готов!"; \
			break; \
		fi; \
		echo "Ожидание... ($$i/10)"; \
		sleep 1; \
	done
	@docker exec obsidian-db pg_isready -U $(DB_USER) > /dev/null 2>&1 || (echo "❌ PostgreSQL не запустился" && exit 1)

db-docker-stop: ## Остановить PostgreSQL Docker
	@docker stop obsidian-db 2>/dev/null || true
	@docker rm obsidian-db 2>/dev/null || true
	@echo "✓ PostgreSQL остановлен"

db-docker-reset: ## Полностью удалить PostgreSQL (с данными!)
	@echo "⚠️  Удаление PostgreSQL контейнера и всех данных..."
	@docker stop obsidian-db 2>/dev/null || true
	@docker rm obsidian-db 2>/dev/null || true
	@docker volume rm obsidian-db-data 2>/dev/null || true
	@echo "✓ PostgreSQL полностью удалён"

db-docker-logs: ## Показать логи PostgreSQL
	docker logs -f obsidian-db

db-check: ## Проверить состояние PostgreSQL
	@./check_postgres.sh

db-migrate: ## Применить миграции
	@echo "Применение миграций..."
	@PGPASSWORD=$(DB_PASSWORD) psql -U $(DB_USER) -d $(DB_NAME) -h localhost -f internal/repository/migrations/001_init_schema.sql
	@echo "✓ Миграции применены"

# Полная настройка для разработки
dev-setup: db-docker-start db-migrate tidy ## Полная настройка (PostgreSQL + миграции + зависимости)
	@echo ""
	@echo "✅ Всё готово для разработки!"
	@echo "Запусти приложение: make dev"
	@echo ""

# Docker
docker-build: ## Собрать Docker образ
	cd deployments/docker && docker-compose build

docker-up: ## Запустить в Docker
	cd deployments/docker && docker-compose up -d

docker-down: ## Остановить Docker
	cd deployments/docker && docker-compose down

docker-logs: ## Показать логи
	cd deployments/docker && docker-compose logs -f

# Утилиты
clean: ## Очистить временные файлы
	rm -f $(BINARY_NAME)
	rm -rf tmp/
	rm -f coverage.out

install-tools: ## Установить dev инструменты
	go install github.com/cosmtrek/air@latest
	go install github.com/go-delve/delve/cmd/dlv@latest

.DEFAULT_GOAL := help
