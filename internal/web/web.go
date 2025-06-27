package web

import (
	"bytes"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"ocuai/internal/auth"
	"ocuai/internal/config"
	"ocuai/internal/events"
	"ocuai/internal/storage"
	"ocuai/internal/streaming"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/gorilla/websocket"
)

// Server представляет веб-сервер
type Server struct {
	config          *config.Config
	storage         *storage.Storage
	eventManager    *events.Manager
	streamingServer *streaming.Server
	webAssets       embed.FS
	upgrader        websocket.Upgrader
	authService     *auth.AuthService
	authHandlers    *auth.AuthHandlers
}

// APIResponse представляет стандартный ответ API
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// CameraRequest представляет запрос для создания/обновления камеры
type CameraRequest struct {
	Name            string  `json:"name"`
	RTSPURL         string  `json:"rtsp_url"`
	Username        string  `json:"username,omitempty"`
	Password        string  `json:"password,omitempty"`
	MotionDetection bool    `json:"motion_detection"`
	AIDetection     bool    `json:"ai_detection"`
	Sensitivity     float32 `json:"sensitivity"`
	RecordMotion    bool    `json:"record_motion"`
	SendTelegram    bool    `json:"send_telegram"`
}

// SystemStats представляет статистику системы
type SystemStats struct {
	CamerasTotal  int `json:"cameras_total"`
	CamerasOnline int `json:"cameras_online"`
	EventsToday   int `json:"events_today"`
	EventsTotal   int `json:"events_total"`
	SystemUptime  int `json:"system_uptime"` // в секундах
}

// New создает новый веб-сервер
func New(cfg *config.Config, storage *storage.Storage, eventManager *events.Manager, streamingServer *streaming.Server, webAssets embed.FS, db *sql.DB) (*Server, error) {
	// Инициализируем сервис авторизации
	authService, err := auth.New(db, cfg.Security.SessionSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth service: %w", err)
	}

	authHandlers := auth.NewHandlers(authService)

	return &Server{
		config:          cfg,
		storage:         storage,
		eventManager:    eventManager,
		streamingServer: streamingServer,
		webAssets:       webAssets,
		authService:     authService,
		authHandlers:    authHandlers,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // В продакшене нужна более строгая проверка
			},
		},
	}, nil
}

// Router создает и настраивает роутер
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Базовые middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS для разработки
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API маршруты
	r.Route("/api", func(r chi.Router) {
		// Публичные маршруты авторизации
		r.Route("/auth", func(r chi.Router) {
			r.Get("/setup", s.authHandlers.CheckSetupHandler)
			r.Post("/register", s.authHandlers.RegisterHandler)
			r.Post("/login", s.authHandlers.LoginHandler)
			r.Post("/logout", s.authHandlers.LogoutHandler)
			r.Get("/status", s.authHandlers.StatusHandler)
		})

		// Защищенные маршруты
		r.Route("/", func(r chi.Router) {
			r.Use(s.authService.RequireAuth())

			r.Get("/health", s.healthHandler)
			r.Get("/stats", s.statsHandler)

			// Камеры
			r.Route("/cameras", func(r chi.Router) {
				r.Get("/", s.getCamerasHandler)
				r.Post("/", s.createCameraHandler)
				r.Get("/{id}", s.getCameraHandler)
				r.Put("/{id}", s.updateCameraHandler)
				r.Delete("/{id}", s.deleteCameraHandler)
				r.Post("/{id}/test", s.testCameraHandler)
			})

			// События
			r.Route("/events", func(r chi.Router) {
				r.Get("/", s.getEventsHandler)
				r.Get("/{id}", s.getEventHandler)
				r.Delete("/{id}", s.deleteEventHandler)
			})

			// Стриминг
			r.Route("/streaming", func(r chi.Router) {
				r.Get("/cameras/{id}/stream", s.streamHandler)
				r.Get("/cameras/{id}/snapshot", s.snapshotHandler)
			})

			// Настройки
			r.Route("/settings", func(r chi.Router) {
				r.Get("/", s.getSettingsHandler)
				r.Put("/", s.updateSettingsHandler)
			})
		})
	})

	// WebSocket для реального времени (защищен)
	r.With(s.authService.RequireAuth()).Get("/ws", s.websocketHandler)

	// Статические файлы веб-интерфейса
	s.setupStaticFiles(r)

	return r
}

// API Handlers

// healthHandler проверка здоровья сервиса
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		},
	})
}

// statsHandler возвращает статистику системы
func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := s.eventManager.GetSystemStats()
	if err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Failed to get system stats: " + err.Error(),
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// getCamerasHandler возвращает список камер
func (s *Server) getCamerasHandler(w http.ResponseWriter, r *http.Request) {
	cameras, err := s.storage.GetCameras()
	if err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Failed to get cameras: " + err.Error(),
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Success: true,
		Data:    cameras,
	})
}

// getCameraHandler возвращает камеру по ID
func (s *Server) getCameraHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	camera, err := s.storage.GetCamera(id)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Failed to get camera: " + err.Error(),
		})
		return
	}

	if camera == nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Camera not found",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Success: true,
		Data:    camera,
	})
}

// createCameraHandler создает новую камеру
func (s *Server) createCameraHandler(w http.ResponseWriter, r *http.Request) {
	var req CameraRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Валидация
	if req.Name == "" || req.RTSPURL == "" {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Name and RTSP URL are required",
		})
		return
	}

	camera := &storage.Camera{
		ID:              generateCameraID(),
		Name:            req.Name,
		RTSPURL:         req.RTSPURL,
		Status:          "offline",
		MotionDetection: req.MotionDetection,
		AIDetection:     req.AIDetection,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.storage.SaveCamera(camera); err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Failed to save camera: " + err.Error(),
		})
		return
	}

	// Добавляем камеру в стриминг сервер
	if err := s.streamingServer.AddCamera(camera.ID, req.RTSPURL); err != nil {
		log.Printf("Failed to add camera to streaming server: %v", err)
	}

	render.JSON(w, r, APIResponse{
		Success: true,
		Data:    camera,
	})
}

// updateCameraHandler обновляет камеру
func (s *Server) updateCameraHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req CameraRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	camera, err := s.storage.GetCamera(id)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Failed to get camera: " + err.Error(),
		})
		return
	}

	if camera == nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Camera not found",
		})
		return
	}

	// Обновляем поля
	camera.Name = req.Name
	camera.RTSPURL = req.RTSPURL
	camera.MotionDetection = req.MotionDetection
	camera.AIDetection = req.AIDetection
	camera.UpdatedAt = time.Now()

	if err := s.storage.SaveCamera(camera); err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Failed to update camera: " + err.Error(),
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Success: true,
		Data:    camera,
	})
}

// deleteCameraHandler удаляет камеру
func (s *Server) deleteCameraHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.storage.DeleteCamera(id); err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Failed to delete camera: " + err.Error(),
		})
		return
	}

	// Удаляем из стриминг сервера
	s.streamingServer.RemoveCamera(id)

	render.JSON(w, r, APIResponse{
		Success: true,
	})
}

// testCameraHandler тестирует подключение к камере
func (s *Server) testCameraHandler(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "id") // Используем для предотвращения ошибки unused variable

	// Здесь должна быть логика тестирования подключения
	// Пока возвращаем заглушку
	render.JSON(w, r, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":     "online",
			"resolution": "1920x1080",
			"fps":        25,
		},
	})
}

// getEventsHandler возвращает список событий
func (s *Server) getEventsHandler(w http.ResponseWriter, r *http.Request) {
	// Параметры пагинации
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	cameraID := r.URL.Query().Get("camera_id")

	limit := 20
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	events, err := s.storage.GetEvents(limit, offset, cameraID)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Failed to get events: " + err.Error(),
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Success: true,
		Data:    events,
	})
}

// getEventHandler возвращает событие по ID
func (s *Server) getEventHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Invalid event ID",
		})
		return
	}

	// Здесь нужно получить событие по ID из storage
	// Пока заглушка
	render.JSON(w, r, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"id":          id,
			"type":        "motion",
			"description": "Motion detected",
		},
	})
}

// deleteEventHandler удаляет событие
func (s *Server) deleteEventHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Invalid event ID",
		})
		return
	}

	// Здесь должна быть логика удаления события
	_ = id

	render.JSON(w, r, APIResponse{
		Success: true,
	})
}

// streamHandler обрабатывает стриминг камеры
func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "id")

	// Проксирование к go2rtc или возвращение RTSP URL
	streamURL := fmt.Sprintf("http://localhost:%d/stream/%s", s.config.Streaming.WebRTCPort, cameraID)

	http.Redirect(w, r, streamURL, http.StatusFound)
}

// snapshotHandler возвращает снапшот с камеры
func (s *Server) snapshotHandler(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "id")

	// Здесь должна быть логика получения снапшота
	// Пока возвращаем заглушку
	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Snapshot not available for camera: " + cameraID))
}

// getSettingsHandler возвращает настройки системы
func (s *Server) getSettingsHandler(w http.ResponseWriter, r *http.Request) {
	settings := map[string]interface{}{
		"ai_enabled":         s.config.AI.Enabled,
		"ai_threshold":       s.config.AI.Threshold,
		"motion_detection":   true,
		"telegram_enabled":   s.config.Telegram.Token != "",
		"notification_hours": s.config.Telegram.NotificationHours,
		"retention_days":     s.config.Storage.RetentionDays,
		"max_video_size_mb":  s.config.Storage.MaxVideoSizeMB,
	}

	render.JSON(w, r, APIResponse{
		Success: true,
		Data:    settings,
	})
}

// updateSettingsHandler обновляет настройки системы
func (s *Server) updateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var settings map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		render.JSON(w, r, APIResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Здесь должна быть логика обновления настроек
	// Пока просто возвращаем успех
	render.JSON(w, r, APIResponse{
		Success: true,
		Data:    settings,
	})
}

// WebSocket handler для реального времени
func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket client connected")

	// Get session for logging
	session, _ := s.authService.GetSession(r)
	if session != nil {
		log.Printf("WebSocket authenticated user: %s", session.Username)
	}

	// Send initial stats
	stats, _ := s.eventManager.GetSystemStats()
	if err := conn.WriteJSON(map[string]interface{}{
		"type": "stats_update",
		"data": stats,
	}); err != nil {
		log.Printf("WebSocket initial write error: %v", err)
		return
	}

	// Create channels for communication
	done := make(chan struct{})

	// Handle incoming messages
	go func() {
		defer close(done)
		for {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				return
			}
			log.Printf("WebSocket received: %v", msg)
		}
	}()

	// Send periodic updates
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			stats, err := s.eventManager.GetSystemStats()
			if err != nil {
				log.Printf("Failed to get stats: %v", err)
				continue
			}

			// Add current timestamp to stats
			stats["timestamp"] = time.Now().Unix()
			stats["current_time"] = time.Now().Format("15:04:05")

			if err := conn.WriteJSON(map[string]interface{}{
				"type": "stats_update",
				"data": stats,
			}); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

// setupStaticFiles настраивает раздачу статических файлов
func (s *Server) setupStaticFiles(r chi.Router) {
	// Try to extract embedded files from different possible paths
	var webFS fs.FS
	var err error

	// Try web/assets first (production build location)
	webFS, err = fs.Sub(s.webAssets, "web/assets")
	if err != nil {
		// Try web/dist as fallback
		webFS, err = fs.Sub(s.webAssets, "web/dist")
		if err != nil {
			log.Printf("Failed to create web filesystem: %v", err)
			// Use local filesystem for development
			localPath := "./web/assets"
			if _, err := os.Stat(localPath); err == nil {
				log.Printf("Using local filesystem at %s", localPath)
				r.Handle("/*", http.FileServer(http.Dir(localPath)))
				return
			}
			// Use fallback handler
			r.Get("/*", s.fallbackHandler)
			return
		}
	}

	// Раздаем статические файлы
	fileServer := http.FileServer(http.FS(webFS))
	r.Handle("/assets/*", fileServer)
	r.Handle("/favicon.ico", fileServer)

	// SPA fallback - все остальные маршруты направляем на index.html
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		file, err := webFS.Open("index.html")
		if err != nil {
			s.fallbackHandler(w, r)
			return
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			s.fallbackHandler(w, r)
			return
		}

		// Читаем содержимое файла в память для подачи клиенту
		content, err := io.ReadAll(file)
		if err != nil {
			s.fallbackHandler(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", stat.ModTime(), bytes.NewReader(content))
	})
}

// fallbackHandler заглушка если нет встроенных файлов
func (s *Server) fallbackHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Ocuai - AI Video Surveillance</title>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0; padding: 40px; background: #0f0f23; color: #cccccc;
            display: flex; flex-direction: column; align-items: center; justify-content: center;
            min-height: 100vh; text-align: center;
        }
        .logo { font-size: 2.5em; margin-bottom: 20px; color: #00ff41; }
        .subtitle { font-size: 1.2em; margin-bottom: 30px; opacity: 0.8; }
        .status { padding: 20px; background: #1a1a2e; border-radius: 8px; margin-bottom: 20px; }
        .api-link { color: #00ff41; text-decoration: none; }
        .api-link:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="logo">🏠 Ocuai</div>
    <div class="subtitle">AI Video Surveillance System</div>
    <div class="status">
        <h3>✅ Сервер запущен</h3>
        <p>Веб-интерфейс будет доступен после сборки фронтенда</p>
        <p>API доступно по адресу: <a href="/api/health" class="api-link">/api/health</a></p>
    </div>
    <div>
        <p><strong>Следующие шаги:</strong></p>
        <ol style="text-align: left; max-width: 400px;">
            <li>Добавьте Telegram токен в конфигурацию</li>
            <li>Подключите IP-камеры через API</li>
            <li>Настройте детекцию движения и AI</li>
        </ol>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// Вспомогательные функции

// generateCameraID генерирует уникальный ID для камеры
func generateCameraID() string {
	return fmt.Sprintf("cam_%d", time.Now().UnixNano())
}
