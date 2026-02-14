-- Create audit_events table for security audit logging
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    actor_type VARCHAR(50) NOT NULL, -- 'user', 'api_key', 'system'
    actor_id UUID,
    resource_type VARCHAR(100) NOT NULL, -- 'project', 'file', 'folder', 'api_key', 'share_link', 'user'
    resource_id UUID,
    action VARCHAR(100) NOT NULL, -- 'create', 'read', 'update', 'delete', 'login', 'logout', 'revoke', etc.
    status VARCHAR(50) NOT NULL, -- 'success', 'failure', 'denied'
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100),
    metadata JSONB, -- Additional context-specific data
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_audit_events_created_at ON audit_events(created_at DESC);
CREATE INDEX idx_audit_events_actor_id ON audit_events(actor_id);
CREATE INDEX idx_audit_events_resource_type_id ON audit_events(resource_type, resource_id);
CREATE INDEX idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX idx_audit_events_status ON audit_events(status);
CREATE INDEX idx_audit_events_request_id ON audit_events(request_id);

-- Composite index for common queries
CREATE INDEX idx_audit_events_actor_created ON audit_events(actor_id, created_at DESC);
