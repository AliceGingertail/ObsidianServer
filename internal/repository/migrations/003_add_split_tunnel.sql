-- 003_add_split_tunnel.sql

-- Split tunnel mode: 'all' (all traffic via VPN), 'include' (only listed), 'exclude' (all except listed)
ALTER TABLE peers ADD COLUMN split_tunnel_mode VARCHAR(20) DEFAULT 'all';

-- Create split tunnel rules table
CREATE TABLE split_tunnel_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    peer_id UUID NOT NULL REFERENCES peers(id) ON DELETE CASCADE,
    rule_type VARCHAR(20) NOT NULL CHECK (rule_type IN ('ip', 'cidr', 'domain')),
    value VARCHAR(255) NOT NULL,
    resolved_ips TEXT[], -- resolved IPs for domains
    description VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(peer_id, value)
);

-- Create indexes
CREATE INDEX idx_split_tunnel_rules_peer_id ON split_tunnel_rules(peer_id);

-- Create trigger for updated_at
CREATE TRIGGER update_split_tunnel_rules_updated_at BEFORE UPDATE ON split_tunnel_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
