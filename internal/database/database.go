package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// NewConnection создает новое соединение с PostgreSQL
func NewConnection(ctx context.Context, config DatabaseConfig) (*pgxpool.Pool, error) {
	// Формируем строку подключения
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.SSLMode,
	)

	// Конфигурируем пул соединений
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Настройки пула
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	// Создаем пул соединений
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем соединение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to PostgreSQL database %s@%s:%d/%s",
		config.User, config.Host, config.Port, config.Database)

	return pool, nil
}

// RunMigrations запускает миграции базы данных
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Создаем таблицу для отслеживания миграций
	migrationTable := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`

	if _, err := pool.Exec(ctx, migrationTable); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	// Список миграций
	migrations := []Migration{
		{
			Version: "001_create_cameras_table",
			SQL: `
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
				CREATE INDEX IF NOT EXISTS idx_cameras_name ON cameras(name);
				CREATE INDEX IF NOT EXISTS idx_cameras_is_active ON cameras(is_active);
				CREATE INDEX IF NOT EXISTS idx_cameras_created_at ON cameras(created_at);

				-- Create updated_at trigger
				CREATE OR REPLACE FUNCTION trigger_set_timestamp()
				RETURNS TRIGGER AS $$
				BEGIN
				  NEW.updated_at = NOW();
				  RETURN NEW;
				END;
				$$ LANGUAGE plpgsql;

				-- Create trigger (drop first to avoid conflicts)
				DROP TRIGGER IF EXISTS set_timestamp ON cameras;
				CREATE TRIGGER set_timestamp
					BEFORE UPDATE ON cameras
					FOR EACH ROW
					EXECUTE PROCEDURE trigger_set_timestamp();

				COMMIT;
			`,
		},
		{
			Version: "002_create_users_table",
			SQL: `
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
			`,
		},
		{
			Version: "003_add_camera_fields",
			SQL: `
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
			`,
		},
	}

	// Применяем миграции
	for _, migration := range migrations {
		if err := applyMigration(ctx, pool, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
		}
	}

	log.Printf("Successfully applied %d migrations", len(migrations))
	return nil
}

// Migration структура миграции
type Migration struct {
	Version string
	SQL     string
}

// applyMigration применяет одну миграцию
func applyMigration(ctx context.Context, pool *pgxpool.Pool, migration Migration) error {
	// Проверяем, применена ли уже миграция
	var exists bool
	err := pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
		migration.Version).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if exists {
		log.Printf("Migration %s already applied, skipping", migration.Version)
		return nil
	}

	// Применяем миграцию в транзакции
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Выполняем SQL миграции
	if _, err := tx.Exec(ctx, migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Записываем информацию о применении миграции
	if _, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1)",
		migration.Version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	log.Printf("Successfully applied migration %s", migration.Version)
	return nil
}
