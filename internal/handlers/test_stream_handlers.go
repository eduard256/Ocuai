package handlers

import (
	"net/http"

	"ocuai/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// TestStreamHandlers хэндлеры для тестирования стримов
type TestStreamHandlers struct {
	testStreamService *services.TestStreamService
}

// NewTestStreamHandlers создает новые хэндлеры для тестирования стримов
func NewTestStreamHandlers(testStreamService *services.TestStreamService) *TestStreamHandlers {
	return &TestStreamHandlers{
		testStreamService: testStreamService,
	}
}

// RegisterRoutes регистрирует маршруты тестирования стримов
func (h *TestStreamHandlers) RegisterRoutes(r chi.Router) {
	// Пока нет функций тестирования стримов
	r.Get("/test-stream", h.TestStreamPlaceholder)
}

// TestStreamPlaceholder заглушка для тестирования стримов
func (h *TestStreamHandlers) TestStreamPlaceholder(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"success": false,
		"message": "Test stream functionality not implemented",
	}

	render.JSON(w, r, response)
}
