-- Migration: 003_add_camera_fields.sql
-- Add missing fields to cameras table

BEGIN;

-- Add missing columns
ALTER TABLE cameras ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'offline';
ALTER TABLE cameras ADD COLUMN IF NOT EXISTS location VARCHAR(255) DEFAULT '';
ALTER TABLE cameras ADD COLUMN IF NOT EXISTS stream_type VARCHAR(50) DEFAULT 'rtsp';
ALTER TABLE cameras ADD COLUMN IF NOT EXISTS resolution VARCHAR(20) DEFAULT '1920x1080';
ALTER TABLE cameras ADD COLUMN IF NOT EXISTS fps INTEGER DEFAULT 30;
ALTER TABLE cameras ADD COLUMN IF NOT EXISTS last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Add constraints
ALTER TABLE cameras ADD CONSTRAINT chk_status_valid CHECK (status IN ('online', 'offline', 'error', 'connecting'));
ALTER TABLE cameras ADD CONSTRAINT chk_fps_positive CHECK (fps > 0 AND fps <= 120);

-- Create indexes for new columns
CREATE INDEX IF NOT EXISTS idx_cameras_status ON cameras(status);
CREATE INDEX IF NOT EXISTS idx_cameras_last_seen ON cameras(last_seen);

COMMIT; 