-- ============================================================
-- MIGRATION: 000001_init_schema.down.sql
-- ============================================================

DROP FUNCTION IF EXISTS is_api_key_valid(UUID);
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TRIGGER IF EXISTS update_files_updated_at ON files;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_clients_updated_at ON clients;

DROP TABLE IF EXISTS share_links;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS folders;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;

-- Break circular FK before dropping
ALTER TABLE clients DROP CONSTRAINT IF EXISTS fk_clients_owner;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS clients;

DROP TYPE IF EXISTS api_key_permission;
DROP TYPE IF EXISTS user_role;

DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
