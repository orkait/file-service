-- Drop audit_events table
DROP INDEX IF EXISTS idx_audit_events_actor_created;
DROP INDEX IF EXISTS idx_audit_events_request_id;
DROP INDEX IF EXISTS idx_audit_events_status;
DROP INDEX IF EXISTS idx_audit_events_event_type;
DROP INDEX IF EXISTS idx_audit_events_resource_type_id;
DROP INDEX IF EXISTS idx_audit_events_actor_id;
DROP INDEX IF EXISTS idx_audit_events_created_at;
DROP TABLE IF EXISTS audit_events;
