package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Storage представляет хранилище данных
type Storage struct {
	db *sql.DB
}

// GetDB возвращает соединение с базой данных
func (s *Storage) GetDB() *sql.DB {
	return s.db
}

// Event представляет событие в системе
type Event struct {
	ID            int       `json:"id"`
	CameraID      string    `json:"camera_id"`
	CameraName    string    `json:"camera_name"`
	Type          string    `json:"type"` // motion, ai_detection
	Description   string    `json:"description"`
	Confidence    float32   `json:"confidence"`
	VideoPath     string    `json:"video_path"`
	ThumbnailPath string    `json:"thumbnail_path"`
	CreatedAt     time.Time `json:"created_at"`
	Processed     bool      `json:"processed"`
}

// Camera представляет камеру в системе
type Camera struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	RTSPURL         string    `json:"rtsp_url"`
	Status          string    `json:"status"` // online, offline, error
	LastSeen        time.Time `json:"last_seen"`
	MotionDetection bool      `json:"motion_detection"`
	AIDetection     bool      `json:"ai_detection"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Settings представляет настройки системы
type Settings struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// New создает новое хранилище
func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &Storage{db: db}

	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return storage, nil
}

// Close закрывает соединение с базой данных
func (s *Storage) Close() error {
	return s.db.Close()
}

// migrate выполняет миграции базы данных
func (s *Storage) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS cameras (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			rtsp_url TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'offline',
			last_seen DATETIME,
			motion_detection BOOLEAN DEFAULT 0,
			ai_detection BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			camera_id TEXT NOT NULL,
			camera_name TEXT NOT NULL,
			type TEXT NOT NULL,
			description TEXT,
			confidence REAL DEFAULT 0,
			video_path TEXT,
			thumbnail_path TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			processed BOOLEAN DEFAULT 0,
			FOREIGN KEY (camera_id) REFERENCES cameras(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_events_camera_id ON events(camera_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_events_type ON events(type)`,
		`CREATE INDEX IF NOT EXISTS idx_cameras_status ON cameras(status)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration query: %w", err)
		}
	}

	return nil
}

// Cameras возвращает все камеры
func (s *Storage) GetCameras() ([]Camera, error) {
	query := `SELECT id, name, rtsp_url, status, last_seen, motion_detection, ai_detection, created_at, updated_at 
			  FROM cameras ORDER BY created_at`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cameras: %w", err)
	}
	defer rows.Close()

	var cameras []Camera
	for rows.Next() {
		var camera Camera
		var lastSeen sql.NullTime

		err := rows.Scan(&camera.ID, &camera.Name, &camera.RTSPURL, &camera.Status,
			&lastSeen, &camera.MotionDetection, &camera.AIDetection,
			&camera.CreatedAt, &camera.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan camera: %w", err)
		}

		if lastSeen.Valid {
			camera.LastSeen = lastSeen.Time
		}

		cameras = append(cameras, camera)
	}

	return cameras, rows.Err()
}

// GetCamera возвращает камеру по ID
func (s *Storage) GetCamera(id string) (*Camera, error) {
	query := `SELECT id, name, rtsp_url, status, last_seen, motion_detection, ai_detection, created_at, updated_at 
			  FROM cameras WHERE id = ?`

	var camera Camera
	var lastSeen sql.NullTime

	err := s.db.QueryRow(query, id).Scan(&camera.ID, &camera.Name, &camera.RTSPURL, &camera.Status,
		&lastSeen, &camera.MotionDetection, &camera.AIDetection,
		&camera.CreatedAt, &camera.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get camera: %w", err)
	}

	if lastSeen.Valid {
		camera.LastSeen = lastSeen.Time
	}

	return &camera, nil
}

// SaveCamera сохраняет или обновляет камеру
func (s *Storage) SaveCamera(camera *Camera) error {
	query := `INSERT OR REPLACE INTO cameras 
			  (id, name, rtsp_url, status, last_seen, motion_detection, ai_detection, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM cameras WHERE id = ?), CURRENT_TIMESTAMP), CURRENT_TIMESTAMP)`

	_, err := s.db.Exec(query, camera.ID, camera.Name, camera.RTSPURL, camera.Status,
		camera.LastSeen, camera.MotionDetection, camera.AIDetection, camera.ID)
	if err != nil {
		return fmt.Errorf("failed to save camera: %w", err)
	}

	return nil
}

// DeleteCamera удаляет камеру
func (s *Storage) DeleteCamera(id string) error {
	_, err := s.db.Exec("DELETE FROM cameras WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete camera: %w", err)
	}
	return nil
}

// UpdateCameraStatus обновляет статус камеры
func (s *Storage) UpdateCameraStatus(id, status string) error {
	query := `UPDATE cameras SET status = ?, last_seen = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := s.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update camera status: %w", err)
	}
	return nil
}

// SaveEvent сохраняет событие
func (s *Storage) SaveEvent(event *Event) error {
	query := `INSERT INTO events (camera_id, camera_name, type, description, confidence, video_path, thumbnail_path, processed)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := s.db.Exec(query, event.CameraID, event.CameraName, event.Type, event.Description,
		event.Confidence, event.VideoPath, event.ThumbnailPath, event.Processed)
	if err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get event id: %w", err)
	}

	event.ID = int(id)
	return nil
}

// GetEvents возвращает события с пагинацией
func (s *Storage) GetEvents(limit, offset int, cameraID string) ([]Event, error) {
	var query string
	var args []interface{}

	if cameraID != "" {
		query = `SELECT id, camera_id, camera_name, type, description, confidence, video_path, thumbnail_path, created_at, processed
				 FROM events WHERE camera_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = []interface{}{cameraID, limit, offset}
	} else {
		query = `SELECT id, camera_id, camera_name, type, description, confidence, video_path, thumbnail_path, created_at, processed
				 FROM events ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = []interface{}{limit, offset}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(&event.ID, &event.CameraID, &event.CameraName, &event.Type,
			&event.Description, &event.Confidence, &event.VideoPath,
			&event.ThumbnailPath, &event.CreatedAt, &event.Processed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetUnprocessedEvents возвращает необработанные события
func (s *Storage) GetUnprocessedEvents() ([]Event, error) {
	query := `SELECT id, camera_id, camera_name, type, description, confidence, video_path, thumbnail_path, created_at, processed
			  FROM events WHERE processed = 0 ORDER BY created_at`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unprocessed events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(&event.ID, &event.CameraID, &event.CameraName, &event.Type,
			&event.Description, &event.Confidence, &event.VideoPath,
			&event.ThumbnailPath, &event.CreatedAt, &event.Processed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// MarkEventProcessed помечает событие как обработанное
func (s *Storage) MarkEventProcessed(id int) error {
	_, err := s.db.Exec("UPDATE events SET processed = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}
	return nil
}

// DeleteOldEvents удаляет старые события
func (s *Storage) DeleteOldEvents(days int) error {
	query := `DELETE FROM events WHERE created_at < datetime('now', '-' || ? || ' days')`
	_, err := s.db.Exec(query, days)
	if err != nil {
		return fmt.Errorf("failed to delete old events: %w", err)
	}
	return nil
}

// GetSetting возвращает настройку по ключу
func (s *Storage) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}

// SetSetting сохраняет настройку
func (s *Storage) SetSetting(key, value string) error {
	query := `INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)`
	_, err := s.db.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}
	return nil
}

// GetStats возвращает статистику
func (s *Storage) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Количество камер
	var cameraCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM cameras").Scan(&cameraCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get camera count: %w", err)
	}
	stats["cameras_total"] = cameraCount

	// Онлайн камеры
	var onlineCameras int
	err = s.db.QueryRow("SELECT COUNT(*) FROM cameras WHERE status = 'online'").Scan(&onlineCameras)
	if err != nil {
		return nil, fmt.Errorf("failed to get online cameras count: %w", err)
	}
	stats["cameras_online"] = onlineCameras

	// События за сегодня
	var todayEvents int
	err = s.db.QueryRow("SELECT COUNT(*) FROM events WHERE DATE(created_at) = DATE('now')").Scan(&todayEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get today events count: %w", err)
	}
	stats["events_today"] = todayEvents

	// Всего событий
	var totalEvents int
	err = s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&totalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total events count: %w", err)
	}
	stats["events_total"] = totalEvents

	return stats, nil
}
