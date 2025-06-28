-- Migration: 002_create_users_table.sql
-- Create users table for authentication

BEGIN;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Add constraints
ALTER TABLE users ADD CONSTRAINT chk_username_not_empty CHECK (trim(username) != '');
ALTER TABLE users ADD CONSTRAINT chk_password_not_empty CHECK (trim(password_hash) != '');
ALTER TABLE users ADD CONSTRAINT chk_role_valid CHECK (role IN ('admin', 'user'));

COMMIT; 