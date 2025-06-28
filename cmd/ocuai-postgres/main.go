package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ocuai/internal/auth"
	"ocuai/internal/config"
	"ocuai/internal/database"
	"ocuai/internal/go2rtc"
	"ocuai/internal/handlers"
	"ocuai/internal/repository"
	"ocuai/internal/services"
	"ocuai/internal/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

var startTime = time.Now()

func main() {
	log.Println("Starting OcuAI Camera Management System with PostgreSQL...")

	// Загружаем конфигурацию (пока не используется, но может понадобиться)
	_, err := config.Load("./data/config.yaml")
	if err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
		// Продолжаем работу с переменными окружения
	}

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Подключаемся к PostgreSQL
	dbConfig := database.DatabaseConfig{
		Host:     getEnv("POSTGRES_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("POSTGRES_USER", "ocuai"),
		Password: getEnv("POSTGRES_PASSWORD", "ocuai123"),
		Database: getEnv("POSTGRES_DB", "ocuai"),
		SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
	}

	db, err := database.NewConnection(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Запускаем миграции
	if err := database.RunMigrations(ctx, db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Инициализируем компоненты
	cameraRepo := repository.NewPostgresCameraRepository(db)

	// Пути для go2rtc
	go2rtcPath := getEnv("GO2RTC_PATH", "./data/go2rtc/bin/go2rtc")
	go2rtcConfig := getEnv("GO2RTC_CONFIG", "./data/go2rtc/go2rtc.yaml")

	cameraService := services.NewCameraService(cameraRepo, go2rtcPath, go2rtcConfig)
	testStreamService := services.NewTestStreamService(cameraService)

	cameraHandlers := handlers.NewCameraHandlers(cameraService)
	testStreamHandlers := handlers.NewTestStreamHandlers(testStreamService)
	systemHandlers := handlers.NewSystemHandlers(cameraService)

	// Инициализируем auth сервис для PostgreSQL
	authService, err := auth.NewPostgres(db, getEnv("SESSION_SECRET", ""))
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v", err)
	}
	authHandlers := auth.NewPostgresHandlers(authService)

	// Инициализируем WebSocket
	wsHub := websocket.NewHub()
	notificationService := websocket.NewNotificationService(wsHub)

	// Запускаем WebSocket hub в горутине
	go wsHub.Run(ctx)
	log.Println("WebSocket hub started")

	// Создаем HTTP роутер
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS настройки
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// WebSocket endpoint - полноценный сервер
	r.Get("/ws", wsHub.ServeWS)

	// Регистрируем API маршруты
	r.Route("/api", func(r chi.Router) {
		// Публичные маршруты авторизации
		r.Route("/auth", func(r chi.Router) {
			r.Get("/setup", authHandlers.CheckSetupHandler)
			r.Post("/register", authHandlers.RegisterHandler)
			r.Post("/login", authHandlers.LoginHandler)
			r.Post("/logout", authHandlers.LogoutHandler)
			r.Get("/status", authHandlers.StatusHandler)
		})

		// Защищенные маршруты
		r.Group(func(r chi.Router) {
			r.Use(authService.RequireAuth())

			// Регистрируем camera и stream маршруты
			cameraHandlers.RegisterRoutes(r)
			testStreamHandlers.RegisterRoutes(r)

			// Системные endpoints
			systemHandlers.RegisterRoutes(r)
		})
	})

	// Статические файлы для NextJS
	staticDir := getEnv("STATIC_DIR", "./web/.next/static")
	nextDir := getEnv("NEXT_DIR", "./web")

	// Serve NextJS static files
	if _, err := os.Stat(staticDir); err == nil {
		fileServer := http.FileServer(http.Dir(staticDir))
		r.Handle("/static/*", http.StripPrefix("/static", fileServer))
		log.Printf("Serving NextJS static files from %s", staticDir)
	}

	// Serve NextJS pages (fallback to index.html for SPA)
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// API routes should return 404, not serve HTML
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Serve specific files if they exist
		indexPath := filepath.Join(nextDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
		} else {
			// Fallback for development
			distPath := filepath.Join(nextDir, "dist", "index.html")
			if _, err := os.Stat(distPath); err == nil {
				http.ServeFile(w, r, distPath)
			} else {
				http.NotFound(w, r)
			}
		}
	})

	// Запускаем HTTP сервер
	server := &http.Server{
		Addr:    ":" + getEnv("PORT", "8080"),
		Handler: r,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("HTTP server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Генерируем начальную конфигурацию go2rtc
	log.Println("Generating initial go2rtc configuration...")
	if err := generateInitialConfig(ctx, cameraService); err != nil {
		log.Printf("Warning: Failed to generate initial go2rtc config: %v", err)
	}

	// Создаем и запускаем go2rtc менеджер
	log.Println("Starting go2rtc streaming server...")
	var go2rtcManager *go2rtc.Manager
	go2rtcManager, err = go2rtc.New("./data/go2rtc")
	if err != nil {
		log.Printf("Warning: Failed to create go2rtc manager: %v", err)
	} else {
		if err := go2rtcManager.Start(); err != nil {
			log.Printf("Warning: Failed to start go2rtc: %v", err)
		} else {
			log.Println("✅ Go2rtc started successfully")
		}
	}

	// Запускаем WebSocket heartbeat для отправки статистики
	go notificationService.StartHeartbeat(ctx, func() interface{} {
		// Получаем реальную статистику из cameraService
		cameraStats, err := cameraService.GetStats(ctx)
		if err != nil {
			log.Printf("Failed to get camera stats: %v", err)
			cameraStats = map[string]interface{}{
				"cameras_total":  0,
				"cameras_active": 0,
				"cameras_online": 0,
			}
		}

		// TODO: Добавить статистику событий из базы данных
		stats := map[string]interface{}{
			"cameras_total":     cameraStats["cameras_total"],
			"cameras_online":    cameraStats["cameras_online"],
			"events_total":      0,                                      // TODO: получать из базы
			"events_today":      0,                                      // TODO: получать из базы
			"system_uptime":     int64(time.Since(startTime).Seconds()), // Реальный uptime в секундах
			"connected_clients": notificationService.GetConnectedClients(),
		}

		return stats
	})

	// Отправляем стартовое уведомление
	notificationService.NotifySystemAlert("System started successfully", "success")

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Отправляем уведомление об остановке
	notificationService.NotifySystemAlert("System shutting down", "warning")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Отменяем контекст для остановки WebSocket
	cancel()

	// Останавливаем go2rtc если он запущен
	if go2rtcManager != nil {
		log.Println("Stopping go2rtc...")
		if err := go2rtcManager.Stop(); err != nil {
			log.Printf("Warning: Failed to stop go2rtc: %v", err)
		} else {
			log.Println("✅ Go2rtc stopped successfully")
		}
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// generateInitialConfig генерирует начальную конфигурацию go2rtc
func generateInitialConfig(ctx context.Context, cameraService *services.CameraService) error {
	// Получаем все камеры из базы
	cameras, err := cameraService.ListCameras(ctx)
	if err != nil {
		return err
	}

	if len(cameras) == 0 {
		log.Println("No cameras found in database, creating default config")
		// Создаем пустую конфигурацию
		defaultConfig := `# Go2RTC Configuration
# Generated automatically from database
# DO NOT EDIT MANUALLY

api:
  listen: ":1984"

streams:
  # No active cameras configured

webrtc:
  listen: ":8555"
  candidates:
    - stun:stun.l.google.com:19302

log:
  level: info
  format: text
`
		configPath := getEnv("GO2RTC_CONFIG", "./data/go2rtc/go2rtc.yaml")
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}

		return os.WriteFile(configPath, []byte(defaultConfig), 0644)
	}

	log.Printf("Found %d cameras in database, generating config", len(cameras))
	return nil // Конфигурация будет сгенерирована автоматически сервисом
}
