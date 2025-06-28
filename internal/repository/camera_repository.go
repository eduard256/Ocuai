package repository

import (
	"context"
	"time"

	"ocuai/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CameraRepository интерфейс для работы с камерами
type CameraRepository interface {
	GetAll(ctx context.Context) ([]models.Camera, error)
	GetByID(ctx context.Context, id string) (*models.Camera, error)
	Update(ctx context.Context, id string, req models.UpdateCameraRequest) error
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

// PostgresCameraRepository реализация репозитория для PostgreSQL
type PostgresCameraRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresCameraRepository создает новый репозиторий камер
func NewPostgresCameraRepository(pool *pgxpool.Pool) CameraRepository {
	return &PostgresCameraRepository{pool: pool}
}

// GetAll возвращает все камеры
func (r *PostgresCameraRepository) GetAll(ctx context.Context) ([]models.Camera, error) {
	query := `
		SELECT id, name, url, status, location, stream_type, resolution, fps, 
		       created_at, updated_at, last_seen
		FROM cameras 
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cameras []models.Camera
	for rows.Next() {
		var camera models.Camera
		err := rows.Scan(
			&camera.ID, &camera.Name, &camera.URL, &camera.Status,
			&camera.Location, &camera.StreamType, &camera.Resolution, &camera.FPS,
			&camera.CreatedAt, &camera.UpdatedAt, &camera.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		cameras = append(cameras, camera)
	}

	return cameras, rows.Err()
}

// GetByID возвращает камеру по ID
func (r *PostgresCameraRepository) GetByID(ctx context.Context, id string) (*models.Camera, error) {
	query := `
		SELECT id, name, url, status, location, stream_type, resolution, fps, 
		       created_at, updated_at, last_seen
		FROM cameras 
		WHERE id = $1`

	var camera models.Camera
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&camera.ID, &camera.Name, &camera.URL, &camera.Status,
		&camera.Location, &camera.StreamType, &camera.Resolution, &camera.FPS,
		&camera.CreatedAt, &camera.UpdatedAt, &camera.LastSeen,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &camera, nil
}

// Update обновляет камеру
func (r *PostgresCameraRepository) Update(ctx context.Context, id string, req models.UpdateCameraRequest) error {
	query := `
		UPDATE cameras 
		SET name = COALESCE($2, name),
		    url = COALESCE($3, url),
		    location = COALESCE($4, location),
		    stream_type = COALESCE($5, stream_type),
		    resolution = COALESCE($6, resolution),
		    fps = COALESCE($7, fps),
		    updated_at = $8
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query,
		id, req.Name, req.URL, req.Location, req.StreamType, req.Resolution, req.FPS,
		time.Now(),
	)

	return err
}

// Delete удаляет камеру
func (r *PostgresCameraRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM cameras WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// UpdateStatus обновляет статус камеры
func (r *PostgresCameraRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE cameras 
		SET status = $2, last_seen = $3, updated_at = $3
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, status, time.Now())
	return err
}
