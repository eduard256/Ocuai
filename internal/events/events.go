package events

import (
	"context"
	"log"
	"sync"
	"time"

	"ocuai/internal/config"
	"ocuai/internal/storage"

	"github.com/robfig/cron/v3"
)

// EventType определяет типы событий
type EventType string

const (
	EventTypeMotion     EventType = "motion"
	EventTypeAI         EventType = "ai_detection"
	EventTypeCameraLost EventType = "camera_lost"
	EventTypeSystemLog  EventType = "system_log"
)

// Event представляет событие в системе
type Event struct {
	Type        EventType              `json:"type"`
	CameraID    string                 `json:"camera_id"`
	CameraName  string                 `json:"camera_name"`
	Description string                 `json:"description"`
	Confidence  float32                `json:"confidence"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
}

// EventHandler функция-обработчик событий
type EventHandler func(Event)

// Manager управляет событиями системы
type Manager struct {
	storage   *storage.Storage
	config    *config.Config
	handlers  map[EventType][]EventHandler
	eventChan chan Event
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	cron      *cron.Cron
}

// New создает новый менеджер событий
func New(storage *storage.Storage, config *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		storage:   storage,
		config:    config,
		handlers:  make(map[EventType][]EventHandler),
		eventChan: make(chan Event, 100),
		ctx:       ctx,
		cancel:    cancel,
		cron:      cron.New(),
	}

	// Запускаем обработчик событий
	manager.wg.Add(1)
	go manager.processEvents()

	// Настраиваем cron задачи
	manager.setupCronJobs()
	manager.cron.Start()

	return manager
}

// Close закрывает менеджер событий
func (m *Manager) Close() {
	m.cron.Stop()
	m.cancel()
	close(m.eventChan)
	m.wg.Wait()
}

// Subscribe подписывается на события определенного типа
func (m *Manager) Subscribe(eventType EventType, handler EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.handlers[eventType] == nil {
		m.handlers[eventType] = make([]EventHandler, 0)
	}
	m.handlers[eventType] = append(m.handlers[eventType], handler)
}

// Emit отправляет событие
func (m *Manager) Emit(event Event) {
	event.Timestamp = time.Now()

	select {
	case m.eventChan <- event:
	case <-m.ctx.Done():
		return
	default:
		log.Printf("Event channel is full, dropping event: %+v", event)
	}
}

// EmitMotionDetected отправляет событие обнаружения движения
func (m *Manager) EmitMotionDetected(cameraID, cameraName string) {
	m.Emit(Event{
		Type:        EventTypeMotion,
		CameraID:    cameraID,
		CameraName:  cameraName,
		Description: "Motion detected",
		Confidence:  1.0,
	})
}

// EmitAIDetection отправляет событие AI детекции
func (m *Manager) EmitAIDetection(cameraID, cameraName, objectClass string, confidence float32, data map[string]interface{}) {
	m.Emit(Event{
		Type:        EventTypeAI,
		CameraID:    cameraID,
		CameraName:  cameraName,
		Description: "Detected: " + objectClass,
		Confidence:  confidence,
		Data:        data,
	})
}

// EmitCameraLost отправляет событие потери камеры
func (m *Manager) EmitCameraLost(cameraID, cameraName string) {
	m.Emit(Event{
		Type:        EventTypeCameraLost,
		CameraID:    cameraID,
		CameraName:  cameraName,
		Description: "Camera connection lost",
		Confidence:  1.0,
	})
}

// EmitSystemLog отправляет системное событие
func (m *Manager) EmitSystemLog(message string) {
	m.Emit(Event{
		Type:        EventTypeSystemLog,
		Description: message,
		Confidence:  1.0,
	})
}

// processEvents обрабатывает события из очереди
func (m *Manager) processEvents() {
	defer m.wg.Done()

	for {
		select {
		case event, ok := <-m.eventChan:
			if !ok {
				return
			}
			m.handleEvent(event)
		case <-m.ctx.Done():
			return
		}
	}
}

// handleEvent обрабатывает отдельное событие
func (m *Manager) handleEvent(event Event) {
	// Сохраняем событие в базу данных (если это не системное событие)
	if event.Type != EventTypeSystemLog {
		dbEvent := &storage.Event{
			CameraID:    event.CameraID,
			CameraName:  event.CameraName,
			Type:        string(event.Type),
			Description: event.Description,
			Confidence:  event.Confidence,
			CreatedAt:   event.Timestamp,
			Processed:   false,
		}

		if err := m.storage.SaveEvent(dbEvent); err != nil {
			log.Printf("Failed to save event to database: %v", err)
		} else {
			log.Printf("Saved event: %s - %s", event.Type, event.Description)
		}
	}

	// Вызываем подписанные обработчики
	m.mu.RLock()
	handlers := m.handlers[event.Type]
	m.mu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Event handler panic: %v", r)
				}
			}()
			h(event)
		}(handler)
	}

	// Логируем событие
	log.Printf("Event processed: %s - %s (Camera: %s)", event.Type, event.Description, event.CameraName)
}

// setupCronJobs настраивает периодические задачи
func (m *Manager) setupCronJobs() {
	// Очистка старых событий (ежедневно в 02:00)
	_, err := m.cron.AddFunc("0 2 * * *", func() {
		if err := m.storage.DeleteOldEvents(m.config.Storage.RetentionDays); err != nil {
			log.Printf("Failed to cleanup old events: %v", err)
		} else {
			log.Printf("Cleaned up events older than %d days", m.config.Storage.RetentionDays)
		}
	})
	if err != nil {
		log.Printf("Failed to add cleanup cron job: %v", err)
	}

	// Проверка статуса камер (каждые 30 минут)
	_, err = m.cron.AddFunc("*/30 * * * *", func() {
		m.checkCameraStatus()
	})
	if err != nil {
		log.Printf("Failed to add camera status check cron job: %v", err)
	}

	// Статистика системы (каждые 5 минут)
	_, err = m.cron.AddFunc("*/5 * * * *", func() {
		stats, err := m.storage.GetStats()
		if err != nil {
			log.Printf("Failed to get system stats: %v", err)
			return
		}

		log.Printf("System stats: %+v", stats)
	})
	if err != nil {
		log.Printf("Failed to add stats cron job: %v", err)
	}
}

// checkCameraStatus проверяет статус камер
func (m *Manager) checkCameraStatus() {
	cameras, err := m.storage.GetCameras()
	if err != nil {
		log.Printf("Failed to get cameras for status check: %v", err)
		return
	}

	for _, camera := range cameras {
		// Если камера была онлайн, но не обновлялась более 2 минут
		if camera.Status == "online" && time.Since(camera.LastSeen) > 2*time.Minute {
			if err := m.storage.UpdateCameraStatus(camera.ID, "offline"); err != nil {
				log.Printf("Failed to update camera status: %v", err)
				continue
			}

			m.EmitCameraLost(camera.ID, camera.Name)
		}
	}
}

// GetRecentEvents возвращает недавние события
func (m *Manager) GetRecentEvents(limit int) ([]storage.Event, error) {
	return m.storage.GetEvents(limit, 0, "")
}

// GetCameraEvents возвращает события для конкретной камеры
func (m *Manager) GetCameraEvents(cameraID string, limit int) ([]storage.Event, error) {
	return m.storage.GetEvents(limit, 0, cameraID)
}

// GetUnprocessedEvents возвращает необработанные события
func (m *Manager) GetUnprocessedEvents() ([]storage.Event, error) {
	return m.storage.GetUnprocessedEvents()
}

// MarkEventProcessed помечает событие как обработанное
func (m *Manager) MarkEventProcessed(eventID int) error {
	return m.storage.MarkEventProcessed(eventID)
}

// GetSystemStats возвращает статистику системы
func (m *Manager) GetSystemStats() (map[string]interface{}, error) {
	return m.storage.GetStats()
}
