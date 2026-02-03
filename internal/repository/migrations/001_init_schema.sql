-- 001_init_schema.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index on email
CREATE INDEX idx_users_email ON users(email);

-- Create peers table
CREATE TABLE peers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name VARCHAR(255) NOT NULL,
    protocol VARCHAR(50) NOT NULL CHECK (protocol IN ('wireguard', 'openvpn')),
    
    -- WireGuard fields
    wg_public_key VARCHAR(255),
    wg_private_key VARCHAR(255),
    wg_preshared VARCHAR(255),
    wg_ip_address VARCHAR(50),
    
    -- OpenVPN fields (future)
    ovpn_certificate TEXT,
    ovpn_private_key TEXT,
    ovpn_ip_address VARCHAR(50),
    
    is_active BOOLEAN DEFAULT TRUE,
    last_seen TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id, device_name)
);

-- Create indexes on peers
CREATE INDEX idx_peers_user_id ON peers(user_id);
CREATE INDEX idx_peers_wg_public_key ON peers(wg_public_key);
CREATE INDEX idx_peers_protocol ON peers(protocol);

-- Create refresh_tokens table
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    device_info TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes on refresh_tokens
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_peers_updated_at BEFORE UPDATE ON peers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
