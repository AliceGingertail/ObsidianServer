-- 005_revert_to_username.sql

-- Добавляем столбец username если его нет
ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(50);

-- Копируем данные из phone в username если phone существует
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'phone') THEN
        UPDATE users SET username = phone WHERE username IS NULL OR username = '';
    END IF;
END $$;

-- Для пользователей без username берем часть email
UPDATE users SET username = SPLIT_PART(email, '@', 1) WHERE (username IS NULL OR username = '') AND email IS NOT NULL;

-- Для остальных генерируем username из id
UPDATE users SET username = 'user_' || LEFT(id::text, 8) WHERE username IS NULL OR username = '';

-- Создаем уникальный индекс
DROP INDEX IF EXISTS idx_users_username;
CREATE UNIQUE INDEX idx_users_username ON users(username);

-- Делаем username обязательным
ALTER TABLE users ALTER COLUMN username SET NOT NULL;

-- Удаляем phone если он существует
ALTER TABLE users DROP COLUMN IF EXISTS phone;
DROP INDEX IF EXISTS idx_users_phone;
