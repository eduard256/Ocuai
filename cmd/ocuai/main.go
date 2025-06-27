package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ocuai/internal/ai"
	"ocuai/internal/config"
	"ocuai/internal/events"
	"ocuai/internal/storage"
	"ocuai/internal/streaming"
	"ocuai/internal/telegram"
	"ocuai/internal/web"
)

// TODO: Fix embed path after build
// go:embed web/dist/*
var webAssets embed.FS

var (
	version = "1.0.0"
	commit  = "dev"
	date    = time.Now().Format("2006-01-02")
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to config file")
		port       = flag.String("port", "", "Server port")
		showVer    = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("Ocuai %s (%s) built on %s\n", version, commit, date)
		os.Exit(0)
	}

	// Инициализация конфигурации
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Переопределение порта из командной строки
	if *port != "" {
		cfg.Server.Port = *port
	}

	// Инициализация базы данных
	store, err := storage.New(cfg.Storage.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Инициализация менеджера событий
	eventManager := events.New(store, cfg)

	// Инициализация AI процессора
	aiProcessor, err := ai.New(cfg.AI)
	if err != nil {
		log.Fatalf("Failed to initialize AI: %v", err)
	}
	defer aiProcessor.Close()

	// Инициализация Telegram бота
	var telegramBot *telegram.Bot
	if cfg.Telegram.Token != "" {
		telegramBot, err = telegram.New(cfg.Telegram, eventManager)
		if err != nil {
			log.Printf("Warning: Failed to initialize Telegram bot: %v", err)
		} else {
			go telegramBot.Start()
			defer telegramBot.Stop()
		}
	}

	// Инициализация стриминг сервера
	streamingServer, err := streaming.New(cfg.Streaming, eventManager, aiProcessor)
	if err != nil {
		log.Fatalf("Failed to initialize streaming server: %v", err)
	}
	defer streamingServer.Close()

	// Запуск стриминг сервера
	go func() {
		if err := streamingServer.Start(); err != nil {
			log.Printf("Streaming server error: %v", err)
		}
	}()

	// Инициализация веб-сервера
	webServer, err := web.New(cfg, store, eventManager, streamingServer, webAssets, store.GetDB())
	if err != nil {
		log.Fatalf("Failed to initialize web server: %v", err)
	}

	// Запуск веб-сервера
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      webServer.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		log.Println("Server stopped")
	}()

	log.Printf("Starting Ocuai v%s on %s", version, server.Addr)
	log.Printf("Web interface: http://%s", server.Addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
