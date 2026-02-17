-- Migration: Add client account lifecycle fields for pause/auto-delete flow

ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active',
  ADD COLUMN IF NOT EXISTS paused_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS scheduled_deletion_at TIMESTAMP;

UPDATE clients
SET status = 'active'
WHERE status IS NULL;

CREATE INDEX IF NOT EXISTS idx_clients_status ON clients(status);
CREATE INDEX IF NOT EXISTS idx_clients_scheduled_deletion_at
  ON clients(scheduled_deletion_at)
  WHERE status = 'paused';
