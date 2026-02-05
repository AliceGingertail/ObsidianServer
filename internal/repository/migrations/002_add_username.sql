-- 002_add_username.sql

-- Добавляем столбец username
ALTER TABLE users ADD COLUMN username VARCHAR(50);

-- Создаем индекс
CREATE UNIQUE INDEX idx_users_username ON users(username);

-- Обновить существ юзеров добавив имя имейла
UPDATE users SET username = SPLIT_PART(email, '@', 1) WHERE username IS NULL;

-- Делаем нот нал
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
