-- Migration: Remove static_url column from assets table
-- This column is redundant since we generate presigned URLs dynamically

ALTER TABLE assets DROP COLUMN IF EXISTS static_url;
