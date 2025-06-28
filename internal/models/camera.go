package models

import "time"

// Camera представляет камеру в системе
type Camera struct {
	ID         string    `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	URL        string    `json:"url" db:"url"`
	Status     string    `json:"status" db:"status"`
	Location   string    `json:"location" db:"location"`
	StreamType string    `json:"stream_type" db:"stream_type"`
	Resolution string    `json:"resolution" db:"resolution"`
	FPS        int       `json:"fps" db:"fps"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
	LastSeen   time.Time `json:"last_seen" db:"last_seen"`
}

// UpdateCameraRequest представляет запрос на обновление камеры
type UpdateCameraRequest struct {
	Name       *string `json:"name,omitempty"`
	URL        *string `json:"url,omitempty"`
	Location   *string `json:"location,omitempty"`
	StreamType *string `json:"stream_type,omitempty"`
	Resolution *string `json:"resolution,omitempty"`
	FPS        *int    `json:"fps,omitempty"`
}
