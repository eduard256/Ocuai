package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"ocuai/internal/models"
	"ocuai/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// CameraHandlers хэндлеры для работы с камерами
type CameraHandlers struct {
	cameraService *services.CameraService
}

// NewCameraHandlers создает новые хэндлеры для камер
func NewCameraHandlers(cameraService *services.CameraService) *CameraHandlers {
	return &CameraHandlers{
		cameraService: cameraService,
	}
}

// RegisterRoutes регистрирует маршруты камер
func (h *CameraHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/cameras", func(r chi.Router) {
		r.Get("/", h.GetCameras)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetCamera)
			r.Put("/", h.UpdateCamera)
			r.Delete("/", h.DeleteCamera)
		})
	})
}

// GetCameras возвращает все камеры
func (h *CameraHandlers) GetCameras(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	cameras, err := h.cameraService.GetAllCameras(ctx)
	if err != nil {
		http.Error(w, "Failed to get cameras", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    cameras,
	}

	render.JSON(w, r, response)
}

// GetCamera возвращает камеру по ID
func (h *CameraHandlers) GetCamera(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := chi.URLParam(r, "id")

	camera, err := h.cameraService.GetCameraByID(ctx, id)
	if err != nil {
		http.Error(w, "Failed to get camera", http.StatusInternalServerError)
		return
	}

	if camera == nil {
		http.Error(w, "Camera not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    camera,
	}

	render.JSON(w, r, response)
}

// UpdateCamera обновляет камеру
func (h *CameraHandlers) UpdateCamera(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := chi.URLParam(r, "id")

	var req models.UpdateCameraRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.cameraService.UpdateCamera(ctx, id, req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Camera updated successfully",
	}

	render.JSON(w, r, response)
}

// DeleteCamera удаляет камеру
func (h *CameraHandlers) DeleteCamera(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := chi.URLParam(r, "id")

	if err := h.cameraService.DeleteCamera(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Camera deleted successfully",
	}

	render.JSON(w, r, response)
}
