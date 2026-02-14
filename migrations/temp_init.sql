-- ============================================================
-- MIGRATION: 000001_init_schema.up.sql
-- Multi-Tenant File Service - Security Hardened
-- ============================================================

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- ENUMS
-- ============================================================

CREATE TYPE user_role AS ENUM ('viewer', 'editor', 'admin');
CREATE TYPE api_key_permission AS ENUM ('read', 'write', 'delete');

-- ============================================================
-- CLIENTS (Tenants)
-- ============================================================

CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clients_owner ON clients(owner_user_id);

-- ============================================================
-- USERS
-- ============================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT check_email_length CHECK (char_length(email) >= 3 AND char_length(email) <= 255)
);

CREATE INDEX idx_users_client ON users(client_id);
CREATE INDEX idx_users_email ON users(email);

ALTER TABLE clients ADD CONSTRAINT fk_clients_owner
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED;

-- ============================================================
-- PROJECTS (1 project = 1 S3 bucket)
-- ============================================================

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    s3_bucket_name VARCHAR(255) NOT NULL UNIQUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_client_project_name UNIQUE (client_id, name),
    CONSTRAINT check_project_name_not_empty CHECK (char_length(name) > 0),
    CONSTRAINT check_project_name_length CHECK (char_length(name) <= 255),
    CONSTRAINT check_bucket_name_format CHECK (s3_bucket_name ~ '^[a-z0-9][a-z0-9-]*[a-z0-9]$')
);

CREATE INDEX idx_projects_client ON projects(client_id);
CREATE INDEX idx_projects_bucket ON projects(s3_bucket_name);
CREATE INDEX idx_projects_default ON projects(client_id, is_default) WHERE is_default = TRUE;

-- ============================================================
-- PROJECT MEMBERS
-- ============================================================

CREATE TABLE project_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role user_role NOT NULL DEFAULT 'viewer',
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    invited_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_project_member UNIQUE (project_id, user_id)
);

CREATE INDEX idx_project_members_project ON project_members(project_id);
CREATE INDEX idx_project_members_user ON project_members(user_id);
CREATE INDEX idx_project_members_role ON project_members(project_id, role);

-- ============================================================
-- FOLDERS
-- ============================================================

CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_folder_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    s3_prefix VARCHAR(1024) NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_folder_path UNIQUE (project_id, s3_prefix),
    CONSTRAINT check_folder_name_not_empty CHECK (char_length(name) > 0),
    CONSTRAINT check_folder_name_valid CHECK (name !~ '[\x00-\x1F\x7F]'),
    CONSTRAINT check_s3_prefix_not_empty CHECK (char_length(s3_prefix) > 0)
);

CREATE INDEX idx_folders_project ON folders(project_id);
CREATE INDEX idx_folders_parent ON folders(parent_folder_id);
CREATE INDEX idx_folders_path ON folders(project_id, s3_prefix);

-- ============================================================
-- FILES
-- ============================================================

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    s3_key VARCHAR(1024) NOT NULL,
    size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(255),
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_file_path UNIQUE (project_id, s3_key),
    CONSTRAINT check_file_name_not_empty CHECK (char_length(name) > 0),
    CONSTRAINT check_file_name_valid CHECK (name !~ '[\x00-\x1F\x7F]'),
    CONSTRAINT check_size_positive CHECK (size_bytes >= 0),
    CONSTRAINT check_size_reasonable CHECK (size_bytes <= 107374182400),
    CONSTRAINT check_s3_key_not_empty CHECK (char_length(s3_key) > 0)
);

CREATE INDEX idx_files_project ON files(project_id);
CREATE INDEX idx_files_folder ON files(folder_id);
CREATE INDEX idx_files_s3_key ON files(s3_key);
CREATE INDEX idx_files_name ON files(project_id, name);
CREATE INDEX idx_files_created_at ON files(project_id, created_at DESC);
CREATE INDEX idx_files_project_folder ON files(project_id, folder_id);

-- ============================================================
-- API KEYS
-- ============================================================

CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(16) NOT NULL,
    permissions api_key_permission[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT check_permissions_not_empty CHECK (array_length(permissions, 1) > 0),
    CONSTRAINT check_key_prefix_format CHECK (key_prefix ~ '^pk_[a-zA-Z0-9]{8}$'),
    CONSTRAINT check_name_not_empty CHECK (char_length(name) > 0)
);

CREATE INDEX idx_api_keys_project ON api_keys(project_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_expiry ON api_keys(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_api_keys_active ON api_keys(project_id, key_hash)
    WHERE revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW());

-- ============================================================
-- SHARE LINKS
-- ============================================================

CREATE TABLE share_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    token VARCHAR(64) NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_token_length CHECK (char_length(token) = 64)
);

CREATE INDEX idx_share_links_file ON share_links(file_id);
CREATE INDEX idx_share_links_token ON share_links(token);

-- ============================================================
-- TRIGGERS
-- ============================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_clients_updated_at BEFORE UPDATE ON clients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_files_updated_at BEFORE UPDATE ON files
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE FUNCTION is_api_key_valid(key_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM api_keys
        WHERE id = key_id
          AND revoked_at IS NULL
          AND (expires_at IS NULL OR expires_at > NOW())
    );
END;
$$ LANGUAGE plpgsql;
