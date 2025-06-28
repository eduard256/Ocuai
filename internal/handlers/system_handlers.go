package handlers

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"ocuai/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// SystemHandlers системные хэндлеры
type SystemHandlers struct {
	cameraService *services.CameraService
	startTime     time.Time
}

// NewSystemHandlers создает новые системные хэндлеры
func NewSystemHandlers(cameraService *services.CameraService) *SystemHandlers {
	return &SystemHandlers{
		cameraService: cameraService,
		startTime:     time.Now(),
	}
}

// RegisterRoutes регистрирует системные маршруты
func (h *SystemHandlers) RegisterRoutes(r chi.Router) {
	r.Get("/stats", h.GetStats)
	r.Get("/events", h.GetEvents) // пустой список событий
}

// GetStats возвращает системную статистику
func (h *SystemHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	cameras, err := h.cameraService.GetAllCameras(ctx)
	if err != nil {
		http.Error(w, "Failed to get cameras", http.StatusInternalServerError)
		return
	}

	onlineCount := 0
	for _, camera := range cameras {
		if camera.Status == "online" {
			onlineCount++
		}
	}

	// Получаем системную информацию
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := int64(time.Since(h.startTime).Seconds())

	stats := map[string]interface{}{
		"cameras_total":  len(cameras),
		"cameras_online": onlineCount,
		"events_today":   0, // пока нет событий
		"events_total":   0, // пока нет событий
		"system_uptime":  uptime,
		"cpu_usage":      0.0,                            // пока не реализовано
		"memory_usage":   float64(m.Alloc) / 1024 / 1024, // MB
		"disk_usage":     0.0,                            // пока не реализовано
	}

	response := map[string]interface{}{
		"success": true,
		"data":    stats,
	}

	render.JSON(w, r, response)
}

// GetEvents возвращает пустой список событий (пока не реализовано)
func (h *SystemHandlers) GetEvents(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"success": true,
		"data":    []interface{}{},
	}

	render.JSON(w, r, response)
}
