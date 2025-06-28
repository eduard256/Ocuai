-- Migration: 001_create_cameras_table.sql
-- Create cameras table for storing camera configurations

BEGIN;

-- Enable uuid-ossp extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create cameras table
CREATE TABLE IF NOT EXISTS cameras (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    url TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_cameras_name ON cameras(name);
CREATE INDEX idx_cameras_is_active ON cameras(is_active);
CREATE INDEX idx_cameras_created_at ON cameras(created_at);

-- Create updated_at trigger
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON cameras
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Add constraints
ALTER TABLE cameras ADD CONSTRAINT chk_name_not_empty CHECK (trim(name) != '');
ALTER TABLE cameras ADD CONSTRAINT chk_url_not_empty CHECK (trim(url) != '');
ALTER TABLE cameras ADD CONSTRAINT chk_url_format CHECK (
    url ~* '^(rtsp|rtmp|http|https|onvif|ffmpeg)://.+'
);

COMMIT; 