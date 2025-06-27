package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config представляет общую конфигурацию приложения
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Storage   StorageConfig   `yaml:"storage"`
	Security  SecurityConfig  `yaml:"security"`
	Telegram  TelegramConfig  `yaml:"telegram"`
	Streaming StreamingConfig `yaml:"streaming"`
	AI        AIConfig        `yaml:"ai"`
	Cameras   []CameraConfig  `yaml:"cameras"`
}

// ServerConfig конфигурация веб-сервера
type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

// StorageConfig конфигурация хранилища
type StorageConfig struct {
	DatabasePath   string `yaml:"database_path"`
	VideoPath      string `yaml:"video_path"`
	RetentionDays  int    `yaml:"retention_days"`
	MaxVideoSizeMB int    `yaml:"max_video_size_mb"`
}

// SecurityConfig конфигурация безопасности
type SecurityConfig struct {
	SessionSecret string `yaml:"session_secret"`
}

// TelegramConfig конфигурация Telegram бота
type TelegramConfig struct {
	Token             string  `yaml:"token"`
	AllowedUsers      []int64 `yaml:"allowed_users"`
	NotificationHours string  `yaml:"notification_hours"`
}

// StreamingConfig конфигурация стриминга
type StreamingConfig struct {
	RTSPPort     int `yaml:"rtsp_port"`
	WebRTCPort   int `yaml:"webrtc_port"`
	BufferSizeKB int `yaml:"buffer_size_kb"`
}

// AIConfig конфигурация AI модуля
type AIConfig struct {
	ModelPath  string   `yaml:"model_path"`
	Enabled    bool     `yaml:"enabled"`
	Threshold  float32  `yaml:"threshold"`
	Classes    []string `yaml:"classes"`
	DeviceType string   `yaml:"device_type"` // cpu, gpu
}

// CameraConfig конфигурация камеры
type CameraConfig struct {
	ID              string  `yaml:"id"`
	Name            string  `yaml:"name"`
	RTSPURL         string  `yaml:"rtsp_url"`
	Username        string  `yaml:"username"`
	Password        string  `yaml:"password"`
	MotionDetection bool    `yaml:"motion_detection"`
	AIDetection     bool    `yaml:"ai_detection"`
	Sensitivity     float32 `yaml:"sensitivity"`
	RecordMotion    bool    `yaml:"record_motion"`
	SendTelegram    bool    `yaml:"send_telegram"`
}

// Load загружает конфигурацию из файла или создает дефолтную
func Load(configPath string) (*Config, error) {
	cfg := defaultConfig()

	// Определяем путь к конфигурации
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Пытаемся загрузить существующий файл
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else {
		// Создаем дефолтный файл конфигурации
		if err := cfg.Save(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	// Переопределяем настройки из переменных окружения
	cfg.overrideFromEnv()

	// Валидация конфигурации
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Save сохраняет конфигурацию в файл
func (c *Config) Save(path string) error {
	// Создаем директорию если не существует
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	if c.Storage.DatabasePath == "" {
		return fmt.Errorf("database path is required")
	}

	if c.Storage.VideoPath == "" {
		return fmt.Errorf("video path is required")
	}

	// Проверяем Telegram конфигурацию если токен задан
	if c.Telegram.Token != "" && len(c.Telegram.AllowedUsers) == 0 {
		return fmt.Errorf("telegram token specified but no allowed users")
	}

	// Создаем необходимые директории
	dirs := []string{
		filepath.Dir(c.Storage.DatabasePath),
		c.Storage.VideoPath,
		filepath.Dir(c.AI.ModelPath),
	}

	for _, dir := range dirs {
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	return nil
}

// overrideFromEnv переопределяет настройки из переменных окружения
func (c *Config) overrideFromEnv() {
	if v := os.Getenv("OCUAI_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("OCUAI_PORT"); v != "" {
		c.Server.Port = v
	}
	if v := os.Getenv("OCUAI_DATABASE_PATH"); v != "" {
		c.Storage.DatabasePath = v
	}
	if v := os.Getenv("OCUAI_VIDEO_PATH"); v != "" {
		c.Storage.VideoPath = v
	}
	if v := os.Getenv("OCUAI_TELEGRAM_TOKEN"); v != "" {
		c.Telegram.Token = v
	}
	if v := os.Getenv("OCUAI_TELEGRAM_USERS"); v != "" {
		users := []int64{}
		for _, userStr := range strings.Split(v, ",") {
			if userID, err := strconv.ParseInt(strings.TrimSpace(userStr), 10, 64); err == nil {
				users = append(users, userID)
			}
		}
		if len(users) > 0 {
			c.Telegram.AllowedUsers = users
		}
	}
	if v := os.Getenv("OCUAI_AI_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			c.AI.Enabled = enabled
		}
	}
}

// defaultConfig возвращает конфигурацию по умолчанию
func defaultConfig() *Config {
	dataDir := getDataDir()

	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: "8080",
		},
		Storage: StorageConfig{
			DatabasePath:   filepath.Join(dataDir, "db", "ocuai.db"),
			VideoPath:      filepath.Join(dataDir, "videos"),
			RetentionDays:  7,
			MaxVideoSizeMB: 50,
		},
		Security: SecurityConfig{
			SessionSecret: "", // Будет сгенерирован автоматически
		},
		Telegram: TelegramConfig{
			Token:             "",
			AllowedUsers:      []int64{},
			NotificationHours: "08:00-22:00",
		},
		Streaming: StreamingConfig{
			RTSPPort:     8554,
			WebRTCPort:   8555,
			BufferSizeKB: 1024,
		},
		AI: AIConfig{
			ModelPath:  filepath.Join(dataDir, "models", "yolov8n.onnx"),
			Enabled:    false,
			Threshold:  0.5,
			Classes:    []string{"person", "car", "truck", "bus", "motorcycle", "bicycle", "dog", "cat"},
			DeviceType: "cpu",
		},
		Cameras: []CameraConfig{},
	}
}

// getDataDir возвращает путь к директории данных
func getDataDir() string {
	if dataDir := os.Getenv("OCUAI_DATA_DIR"); dataDir != "" {
		return dataDir
	}

	// Используем относительный путь к проекту для более предсказуемого поведения
	return "./data"
}

// getDefaultConfigPath возвращает путь к файлу конфигурации по умолчанию
func getDefaultConfigPath() string {
	return filepath.Join(getDataDir(), "config.yaml")
}

// IsNotificationTimeAllowed проверяет, разрешено ли отправлять уведомления в текущее время
func (c *Config) IsNotificationTimeAllowed() bool {
	if c.Telegram.NotificationHours == "" {
		return true
	}

	parts := strings.Split(c.Telegram.NotificationHours, "-")
	if len(parts) != 2 {
		return true
	}

	startTime, err1 := time.Parse("15:04", strings.TrimSpace(parts[0]))
	endTime, err2 := time.Parse("15:04", strings.TrimSpace(parts[1]))

	if err1 != nil || err2 != nil {
		return true
	}

	now := time.Now()
	currentTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC)
	startTime = time.Date(0, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
	endTime = time.Date(0, 1, 1, endTime.Hour(), endTime.Minute(), 0, 0, time.UTC)

	if startTime.Before(endTime) {
		return currentTime.After(startTime) && currentTime.Before(endTime)
	} else {
		// Переход через полночь
		return currentTime.After(startTime) || currentTime.Before(endTime)
	}
}
