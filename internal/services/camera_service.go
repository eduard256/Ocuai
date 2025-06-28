package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ocuai/internal/models"
	"ocuai/internal/repository"

	"gopkg.in/yaml.v3"
)

// CameraService сервис для работы с камерами
type CameraService struct {
	repo         repository.CameraRepository
	go2rtcPath   string
	go2rtcConfig string
}

// NewCameraService создает новый сервис камер
func NewCameraService(repo repository.CameraRepository, go2rtcPath, go2rtcConfig string) *CameraService {
	return &CameraService{
		repo:         repo,
		go2rtcPath:   go2rtcPath,
		go2rtcConfig: go2rtcConfig,
	}
}

// GetAllCameras возвращает все камеры
func (s *CameraService) GetAllCameras(ctx context.Context) ([]models.Camera, error) {
	return s.repo.GetAll(ctx)
}

// GetCameraByID возвращает камеру по ID
func (s *CameraService) GetCameraByID(ctx context.Context, id string) (*models.Camera, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateCamera обновляет камеру
func (s *CameraService) UpdateCamera(ctx context.Context, id string, req models.UpdateCameraRequest) error {
	// Проверяем что камера существует
	camera, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get camera: %w", err)
	}
	if camera == nil {
		return fmt.Errorf("camera not found")
	}

	// Обновляем камеру
	if err := s.repo.Update(ctx, id, req); err != nil {
		return fmt.Errorf("failed to update camera: %w", err)
	}

	// Обновляем go2rtc конфигурацию
	if err := s.updateGo2rtcConfig(ctx); err != nil {
		log.Printf("Warning: Failed to update go2rtc config: %v", err)
	}

	return nil
}

// DeleteCamera удаляет камеру
func (s *CameraService) DeleteCamera(ctx context.Context, id string) error {
	// Проверяем что камера существует
	camera, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get camera: %w", err)
	}
	if camera == nil {
		return fmt.Errorf("camera not found")
	}

	// Удаляем камеру
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete camera: %w", err)
	}

	// Обновляем go2rtc конфигурацию
	if err := s.updateGo2rtcConfig(ctx); err != nil {
		log.Printf("Warning: Failed to update go2rtc config: %v", err)
	}

	return nil
}

// UpdateCameraStatus обновляет статус камеры
func (s *CameraService) UpdateCameraStatus(ctx context.Context, id string, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

// updateGo2rtcConfig обновляет конфигурацию go2rtc
func (s *CameraService) updateGo2rtcConfig(ctx context.Context) error {
	cameras, err := s.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cameras: %w", err)
	}

	// Создаем конфигурацию go2rtc
	config := map[string]interface{}{
		"api": map[string]interface{}{
			"listen": ":1984",
		},
		"rtsp": map[string]interface{}{
			"listen": ":8554",
		},
		"webrtc": map[string]interface{}{
			"listen": ":8555",
		},
		"streams": make(map[string]interface{}),
	}

	// Добавляем камеры в конфигурацию
	streams := config["streams"].(map[string]interface{})
	for _, camera := range cameras {
		streams[camera.ID] = camera.URL
	}

	// Создаем директорию если не существует
	configDir := filepath.Dir(s.go2rtcConfig)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Записываем конфигурацию
	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(s.go2rtcConfig, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ListCameras возвращает все камеры (алиас для GetAllCameras)
func (s *CameraService) ListCameras(ctx context.Context) ([]models.Camera, error) {
	return s.GetAllCameras(ctx)
}

// GetStats возвращает базовую статистику камер
func (s *CameraService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	cameras, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cameras: %w", err)
	}

	onlineCount := 0
	for _, camera := range cameras {
		if camera.Status == "online" {
			onlineCount++
		}
	}

	stats := map[string]interface{}{
		"cameras_total":  len(cameras),
		"cameras_online": onlineCount,
	}

	return stats, nil
}
