-- 004_add_phone.sql

-- Добавляем столбец phone
ALTER TABLE users ADD COLUMN phone VARCHAR(20);

-- Создаем уникальный индекс на phone
CREATE UNIQUE INDEX idx_users_phone ON users(phone);

-- Делаем email опциональным
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;

-- Удаляем уникальное ограничение с email (оставляем индекс)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;

-- Удаляем username (больше не нужен)
DROP INDEX IF EXISTS idx_users_username;
ALTER TABLE users DROP COLUMN IF EXISTS username;

-- Для существующих пользователей временно ставим phone = email
UPDATE users SET phone = COALESCE(email, id::text) WHERE phone IS NULL;

-- Делаем phone обязательным
ALTER TABLE users ALTER COLUMN phone SET NOT NULL;
