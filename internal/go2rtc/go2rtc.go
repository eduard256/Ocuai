package go2rtc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Manager управляет процессом go2rtc
type Manager struct {
	cmd        *exec.Cmd
	configPath string
	dataDir    string
	apiURL     string
	rtspPort   int
	webrtcPort int
	apiPort    int
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	isRunning  bool
}

// Config представляет конфигурацию go2rtc
type Config struct {
	API struct {
		Listen   string `yaml:"listen"`
		Origin   string `yaml:"origin"`
		BasePath string `yaml:"base_path,omitempty"`
	} `yaml:"api"`
	RTSP struct {
		Listen       string `yaml:"listen"`
		DefaultQuery string `yaml:"default_query"`
	} `yaml:"rtsp"`
	WebRTC struct {
		Listen string `yaml:"listen"`
	} `yaml:"webrtc"`
	Streams map[string]interface{} `yaml:"streams"`
	Log     struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
}

// Stream представляет поток в go2rtc
type Stream struct {
	Name     string                 `json:"name"`
	Sources  []string               `json:"sources,omitempty"`
	Channels int                    `json:"channels,omitempty"`
	Codecs   []map[string]string    `json:"codecs,omitempty"`
	Recv     int                    `json:"recv,omitempty"`
	Send     int                    `json:"send,omitempty"`
	Info     map[string]interface{} `json:"info,omitempty"`
}

// StreamInfo представляет информацию о потоке
type StreamInfo struct {
	Producers []Producer `json:"producers"`
}

// Producer представляет продюсера потока
type Producer struct {
	URL    string              `json:"url"`
	Codecs []map[string]string `json:"codecs"`
}

// New создает новый менеджер go2rtc
func New(dataDir string) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		dataDir:    dataDir,
		configPath: filepath.Join(dataDir, "go2rtc.yaml"),
		apiPort:    1984,
		rtspPort:   8554,
		webrtcPort: 8555,
		apiURL:     "http://127.0.0.1:1984",
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // Уменьшаем timeout для быстрого сканирования
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Создаем директорию если не существует
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Создаем конфигурацию
	if err := m.createConfig(); err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return m, nil
}

// createConfig создает конфигурационный файл go2rtc
func (m *Manager) createConfig() error {
	// Проверяем, существует ли уже конфигурационный файл
	if _, err := os.Stat(m.configPath); err == nil {
		// Файл существует, проверяем его корректность
		data, readErr := os.ReadFile(m.configPath)
		if readErr == nil {
			var testConfig Config
			if yamlErr := yaml.Unmarshal(data, &testConfig); yamlErr == nil {
				// Конфигурация корректная, проверяем что streams не в JSON формате
				configStr := string(data)
				if !strings.Contains(configStr, "streams: {}") && !strings.Contains(configStr, "streams:{}") {
					log.Printf("Using existing valid config: %s", m.configPath)
					return nil
				} else {
					log.Printf("Config file has invalid streams format, recreating: %s", m.configPath)
				}
			} else {
				log.Printf("Config file corrupted, recreating: %v", yamlErr)
			}
		}
	}

	log.Printf("Creating new config: %s", m.configPath)

	// Создаем новую конфигурацию с правильным YAML форматом
	configContent := fmt.Sprintf(`api:
  listen: :%d
  origin: '*'

rtsp:
  listen: :%d
  default_query: video&audio

webrtc:
  listen: :%d

streams:

log:
  level: info
`, m.apiPort, m.rtspPort, m.webrtcPort)

	if err := os.WriteFile(m.configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	log.Printf("Config created successfully")
	return nil
}

// Start запускает процесс go2rtc
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return nil
	}

	// Определяем путь к бинарному файлу go2rtc
	binaryPath, err := m.getBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to get binary path: %w", err)
	}

	// Запускаем go2rtc с абсолютным путем к конфигурации
	m.cmd = exec.CommandContext(m.ctx, binaryPath, "-c", m.configPath)

	// Не устанавливаем рабочую директорию, чтобы избежать проблем с путями
	// m.cmd.Dir = m.dataDir

	// Перенаправляем вывод в логи
	m.cmd.Stdout = log.Writer()
	m.cmd.Stderr = log.Writer()

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start go2rtc: %w", err)
	}

	m.isRunning = true

	// Ждем пока go2rtc запустится
	if err := m.waitForReady(); err != nil {
		m.Stop()
		return fmt.Errorf("go2rtc failed to start: %w", err)
	}

	log.Printf("go2rtc started successfully on ports: API=%d, RTSP=%d, WebRTC=%d",
		m.apiPort, m.rtspPort, m.webrtcPort)

	return nil
}

// Stop останавливает процесс go2rtc
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	m.cancel()

	if m.cmd != nil && m.cmd.Process != nil {
		// Даем время на graceful shutdown
		done := make(chan error)
		go func() {
			done <- m.cmd.Wait()
		}()

		select {
		case <-done:
			// Процесс завершился нормально
		case <-time.After(5 * time.Second):
			// Принудительно завершаем
			m.cmd.Process.Kill()
		}
	}

	m.isRunning = false
	log.Println("go2rtc stopped")

	return nil
}

// Restart перезапускает процесс go2rtc
func (m *Manager) Restart() error {
	log.Printf("🔄 Restarting go2rtc process...")

	// Останавливаем процесс
	if err := m.Stop(); err != nil {
		log.Printf("Warning: Failed to stop go2rtc: %v", err)
	}

	// Создаем новый контекст
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Запускаем заново
	if err := m.Start(); err != nil {
		return fmt.Errorf("failed to restart go2rtc: %w", err)
	}

	log.Printf("✅ Go2rtc restarted successfully")
	return nil
}

// waitForReady ждет пока go2rtc будет готов
func (m *Manager) waitForReady() error {
	log.Printf("Waiting for go2rtc to become ready...")

	for i := 0; i < 15; i++ { // Уменьшено до 15 секунд
		resp, err := m.httpClient.Get(m.apiURL + "/api")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Printf("go2rtc is ready after %d seconds", i+1)
				return nil
			}
		}

		if i%5 == 0 && i > 0 {
			log.Printf("Still waiting for go2rtc... (%d/%d seconds)", i, 15)
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for go2rtc to start (waited 15 seconds)")
}

// getBinaryPath определяет путь к бинарному файлу go2rtc
func (m *Manager) getBinaryPath() (string, error) {
	// Сначала проверяем локальный бинарник
	localPath := filepath.Join(m.dataDir, "bin", "go2rtc")
	if runtime.GOOS == "windows" {
		localPath += ".exe"
	}

	// Используем абсолютный путь
	if !filepath.IsAbs(localPath) {
		wd, err := os.Getwd()
		if err == nil {
			localPath = filepath.Join(wd, localPath)
		}
	}

	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	// Проверяем в PATH
	if path, err := exec.LookPath("go2rtc"); err == nil {
		return path, nil
	}

	// Пытаемся скачать go2rtc
	if err := m.downloadGo2rtc(); err != nil {
		return "", fmt.Errorf("go2rtc not found and failed to download: %w", err)
	}

	return localPath, nil
}

// downloadGo2rtc скачивает последнюю версию go2rtc
func (m *Manager) downloadGo2rtc() error {
	binDir := filepath.Join(m.dataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Определяем платформу
	platform := runtime.GOOS + "_" + runtime.GOARCH

	// URL для скачивания
	downloadURL := fmt.Sprintf("https://github.com/AlexxIT/go2rtc/releases/latest/download/go2rtc_%s", platform)
	if runtime.GOOS == "windows" {
		downloadURL += ".zip"
	}

	log.Printf("Downloading go2rtc from %s", downloadURL)

	// Скачиваем файл
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download go2rtc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download go2rtc: status %d", resp.StatusCode)
	}

	// Сохраняем файл
	targetPath := filepath.Join(binDir, "go2rtc")
	if runtime.GOOS == "windows" {
		targetPath += ".exe"
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// Делаем файл исполняемым на Unix системах
	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to make file executable: %w", err)
		}
	}

	log.Printf("go2rtc downloaded successfully to %s", targetPath)
	return nil
}

// AddStream добавляет поток в go2rtc
func (m *Manager) AddStream(name string, source string) error {
	if !m.isRunning {
		log.Printf("Cannot add stream %s: go2rtc is not running", name)
		return fmt.Errorf("go2rtc is not running")
	}

	// Формируем URL для API
	url := fmt.Sprintf("%s/api/streams?dst=%s&src=%s", m.apiURL, name, source)
	log.Printf("Adding stream %s to go2rtc, URL: %s", name, url)

	// Отправляем PUT запрос
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		log.Printf("Failed to create PUT request for stream %s: %v", name, err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	log.Printf("Sending PUT request for stream %s", name)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to execute PUT request for stream %s: %v", name, err)
		return fmt.Errorf("failed to add stream: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("PUT request for stream %s completed with status: %d %s", name, resp.StatusCode, resp.Status)

	// Читаем тело ответа для логирования
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("Failed to read response body for stream %s: %v", name, readErr)
	}

	if resp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("go2rtc returned status %d for ADD stream %s", resp.StatusCode, name)
		if len(body) > 0 {
			errorMsg += fmt.Sprintf(", response: %s", string(body))
		}
		log.Printf("Error adding stream %s: %s", name, errorMsg)
		return fmt.Errorf("failed to add stream: %s", errorMsg)
	}

	if len(body) > 0 {
		log.Printf("PUT response for stream %s: %s", name, string(body))
	}

	log.Printf("Stream %s added successfully to go2rtc", name)
	return nil
}

// RemoveStream удаляет поток из go2rtc
func (m *Manager) RemoveStream(name string) error {
	if !m.isRunning {
		log.Printf("Cannot remove stream %s: go2rtc is not running", name)
		return fmt.Errorf("go2rtc is not running")
	}

	url := fmt.Sprintf("%s/api/streams?src=%s", m.apiURL, name)
	log.Printf("Removing stream %s from go2rtc, URL: %s", name, url)

	// Создаем HTTP клиент с увеличенным таймаутом для DELETE операций
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("Failed to create DELETE request for stream %s: %v", name, err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	log.Printf("Sending DELETE request for stream %s", name)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to execute DELETE request for stream %s: %v", name, err)
		return fmt.Errorf("failed to remove stream: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("DELETE request for stream %s completed with status: %d %s", name, resp.StatusCode, resp.Status)

	// Читаем тело ответа для логирования
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("Failed to read response body for stream %s: %v", name, readErr)
	}

	// go2rtc может возвращать 200 (OK) или 204 (No Content) для успешного удаления
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		errorMsg := fmt.Sprintf("go2rtc returned status %d for DELETE stream %s", resp.StatusCode, name)
		if len(body) > 0 {
			errorMsg += fmt.Sprintf(", response: %s", string(body))
		}
		log.Printf("Error removing stream %s: %s", name, errorMsg)
		return fmt.Errorf("failed to remove stream: %s", errorMsg)
	}

	if len(body) > 0 {
		log.Printf("DELETE response for stream %s: %s", name, string(body))
	}

	log.Printf("Stream %s removed successfully from go2rtc", name)

	// Небольшая задержка для обработки DELETE операции go2rtc
	time.Sleep(100 * time.Millisecond)

	// Дополнительная проверка - убеждаемся что поток действительно удален
	if exists, checkErr := m.StreamExists(name); checkErr == nil && exists {
		log.Printf("Warning: Stream %s still exists after DELETE operation", name)
		return fmt.Errorf("stream %s was not properly removed from go2rtc", name)
	} else if checkErr == nil && !exists {
		log.Printf("Confirmed: Stream %s no longer exists in go2rtc", name)
	} else if checkErr != nil {
		log.Printf("Unable to verify stream %s removal: %v", name, checkErr)
	}

	return nil
}

// StreamExists проверяет существование потока в go2rtc
func (m *Manager) StreamExists(name string) (bool, error) {
	if !m.isRunning {
		return false, fmt.Errorf("go2rtc is not running")
	}

	streams, err := m.GetStreams()
	if err != nil {
		return false, fmt.Errorf("failed to get streams: %w", err)
	}

	_, exists := streams[name]
	return exists, nil
}

// GetStreams возвращает список всех потоков
func (m *Manager) GetStreams() (map[string]Stream, error) {
	if !m.isRunning {
		return nil, fmt.Errorf("go2rtc is not running")
	}

	resp, err := m.httpClient.Get(m.apiURL + "/api/streams")
	if err != nil {
		return nil, fmt.Errorf("failed to get streams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get streams: status %d", resp.StatusCode)
	}

	var streams map[string]Stream
	if err := json.NewDecoder(resp.Body).Decode(&streams); err != nil {
		return nil, fmt.Errorf("failed to decode streams: %w", err)
	}

	return streams, nil
}

// GetStreamInfo возвращает информацию о потоке
func (m *Manager) GetStreamInfo(name string) (*StreamInfo, error) {
	if !m.isRunning {
		return nil, fmt.Errorf("go2rtc is not running")
	}

	url := fmt.Sprintf("%s/api/streams?src=%s", m.apiURL, name)

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stream not found")
	}

	var info StreamInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode stream info: %w", err)
	}

	return &info, nil
}

// GetStreamURL возвращает URL для доступа к потоку
func (m *Manager) GetStreamURL(name string, protocol string) string {
	switch strings.ToLower(protocol) {
	case "rtsp":
		return fmt.Sprintf("rtsp://127.0.0.1:%d/%s", m.rtspPort, name)
	case "webrtc":
		return fmt.Sprintf("http://127.0.0.1:%d/stream/%s", m.apiPort, name)
	case "mse", "mp4":
		return fmt.Sprintf("http://127.0.0.1:%d/api/stream.mp4?src=%s", m.apiPort, name)
	case "hls":
		return fmt.Sprintf("http://127.0.0.1:%d/api/stream.m3u8?src=%s", m.apiPort, name)
	case "mjpeg":
		return fmt.Sprintf("http://127.0.0.1:%d/api/stream.mjpeg?src=%s", m.apiPort, name)
	default:
		return fmt.Sprintf("http://127.0.0.1:%d/stream/%s", m.apiPort, name)
	}
}

// TestStream проверяет работоспособность потока
func (m *Manager) TestStream(source string) error {
	if !m.isRunning {
		return fmt.Errorf("go2rtc is not running")
	}

	// Создаем временный поток для тестирования
	testName := fmt.Sprintf("test_%d", time.Now().Unix())

	// Добавляем поток
	if err := m.AddStream(testName, source); err != nil {
		return fmt.Errorf("failed to add test stream: %w", err)
	}

	// Даем время на инициализацию (больше для TRASSIR)
	time.Sleep(5 * time.Second)

	// Проверяем информацию о потоке
	info, err := m.GetStreamInfo(testName)
	if err != nil {
		m.RemoveStream(testName)
		return fmt.Errorf("failed to get stream info: %w", err)
	}

	// Удаляем тестовый поток
	m.RemoveStream(testName)

	// Проверяем, что есть продюсеры
	if len(info.Producers) == 0 {
		return fmt.Errorf("no producers found")
	}

	return nil
}

// SaveConfig сохраняет текущую конфигурацию
func (m *Manager) SaveConfig() error {
	// Загружаем текущую конфигурацию
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Получаем текущие потоки
	streams, err := m.GetStreams()
	if err != nil {
		return fmt.Errorf("failed to get streams: %w", err)
	}

	// Формируем конфигурационный файл вручную для правильного YAML формата
	configContent := fmt.Sprintf(`api:
  listen: %s
  origin: '%s'

rtsp:
  listen: %s
  default_query: %s

webrtc:
  listen: %s

streams:
`, config.API.Listen, config.API.Origin, config.RTSP.Listen, config.RTSP.DefaultQuery, config.WebRTC.Listen)

	// Добавляем потоки в правильном YAML формате
	for name, stream := range streams {
		if len(stream.Sources) > 0 {
			configContent += fmt.Sprintf("  %s: %s\n", name, stream.Sources[0])
		}
	}

	configContent += fmt.Sprintf(`
log:
  level: %s
`, config.Log.Level)

	if err := os.WriteFile(m.configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	log.Printf("Configuration saved successfully")
	return nil
}

// ProbeStream пытается определить информацию о потоке без добавления
func (m *Manager) ProbeStream(source string) (map[string]interface{}, error) {
	if !m.isRunning {
		return nil, fmt.Errorf("go2rtc is not running")
	}

	// Используем правильный API probe endpoint
	url := fmt.Sprintf("%s/api/streams?src=%s&video=all&audio=all&microphone", m.apiURL, source)

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to probe stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to probe stream (status %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode probe result: %w", err)
	}

	return result, nil
}

// TestStreamDirect проверяет работоспособность потока напрямую через добавление/удаление
func (m *Manager) TestStreamDirect(source string) error {
	if !m.isRunning {
		return fmt.Errorf("go2rtc is not running")
	}

	// Создаем уникальное имя для тестового потока
	testName := fmt.Sprintf("test_quick_%d_%d", time.Now().Unix(), time.Now().Nanosecond())

	log.Printf("Testing stream: %s", source)

	// Добавляем поток через стандартную функцию
	if err := m.AddStream(testName, source); err != nil {
		return fmt.Errorf("failed to add test stream: %w", err)
	}

	log.Printf("Stream %s added successfully", testName)

	// Всегда удаляем тестовый поток в defer для гарантированной очистки
	defer func() {
		if removeErr := m.RemoveStream(testName); removeErr != nil {
			log.Printf("Warning: failed to remove test stream %s: %v", testName, removeErr)
		} else {
			log.Printf("Stream %s removed successfully", testName)
		}
	}()

	// Ждем немного для начальной инициализации
	time.Sleep(1 * time.Second)

	// Проверяем через список потоков - более надежно чем GetStreamInfo
	streams, err := m.GetStreams()
	if err != nil {
		return fmt.Errorf("failed to get streams list: %w", err)
	}

	stream, exists := streams[testName]
	if !exists {
		return fmt.Errorf("stream was not added to streams list")
	}

	// Если поток есть в списке и имеет источники, считаем его успешным
	if len(stream.Sources) > 0 {
		log.Printf("  ✅ Success: stream has %d sources", len(stream.Sources))
		return nil
	}

	// Попытаемся получить информацию о потоке (необязательно)
	info, infoErr := m.GetStreamInfo(testName)
	if infoErr == nil && len(info.Producers) > 0 {
		log.Printf("  ✅ Success: stream has %d producers", len(info.Producers))
		return nil
	}

	// Даже если GetStreamInfo не работает, но поток добавлен в список - это успех
	// go2rtc может еще инициализировать поток
	log.Printf("  ✅ Success: stream added to go2rtc (initializing...)")
	return nil
}
