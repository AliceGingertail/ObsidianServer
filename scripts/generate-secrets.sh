#!/bin/bash

# Скрипт для генерации JWT секретов

echo "Генерация JWT секретов..."
echo ""

ACCESS_SECRET=$(openssl rand -base64 32)
REFRESH_SECRET=$(openssl rand -base64 32)

echo "Скопируйте эти значения в ваш .env файл:"
echo ""
echo "JWT_ACCESS_SECRET=$ACCESS_SECRET"
echo "JWT_REFRESH_SECRET=$REFRESH_SECRET"
echo ""
echo "Или добавьте их автоматически:"
echo ""
echo "echo 'JWT_ACCESS_SECRET=$ACCESS_SECRET' >> .env"
echo "echo 'JWT_REFRESH_SECRET=$REFRESH_SECRET' >> .env"
