package websocket

import (
	"context"
	"log"
	"time"
)

// NotificationService управляет уведомлениями через WebSocket
type NotificationService struct {
	hub *Hub
}

// NewNotificationService создает новый сервис уведомлений
func NewNotificationService(hub *Hub) *NotificationService {
	return &NotificationService{
		hub: hub,
	}
}

// NotifyCameraStatus отправляет уведомление об изменении статуса камеры
func (s *NotificationService) NotifyCameraStatus(cameraID, cameraName, status string) {
	message := &Message{
		Type:       "camera_status",
		CameraID:   cameraID,
		CameraName: cameraName,
		Status:     status,
	}
	s.hub.Broadcast(message)
	log.Printf("Broadcasted camera status: %s - %s", cameraName, status)
}

// NotifyMotionDetected отправляет уведомление об обнаружении движения
func (s *NotificationService) NotifyMotionDetected(cameraID, cameraName string) {
	message := &Message{
		Type:       "motion_detected",
		CameraID:   cameraID,
		CameraName: cameraName,
		Message:    "Motion detected",
	}
	s.hub.Broadcast(message)
	log.Printf("Broadcasted motion detection: %s", cameraName)
}

// NotifyAIDetection отправляет уведомление об AI детекции
func (s *NotificationService) NotifyAIDetection(cameraID, cameraName, objectClass string, confidence float64) {
	message := &Message{
		Type:        "ai_detection",
		CameraID:    cameraID,
		CameraName:  cameraName,
		ObjectClass: objectClass,
		Confidence:  confidence,
	}
	s.hub.Broadcast(message)
	log.Printf("Broadcasted AI detection: %s detected %s (%.2f%%)", cameraName, objectClass, confidence*100)
}

// NotifyNewEvent отправляет уведомление о новом событии
func (s *NotificationService) NotifyNewEvent(event interface{}) {
	message := &Message{
		Type:  "new_event",
		Event: event,
	}
	s.hub.Broadcast(message)
	log.Printf("Broadcasted new event")
}

// NotifySystemAlert отправляет системное уведомление
func (s *NotificationService) NotifySystemAlert(alertMessage, level string) {
	message := &Message{
		Type:    "system_alert",
		Message: alertMessage,
		Level:   level,
	}
	s.hub.Broadcast(message)
	log.Printf("Broadcasted system alert: %s (%s)", alertMessage, level)
}

// NotifyCameraRemoved отправляет уведомление об удалении камеры
func (s *NotificationService) NotifyCameraRemoved(cameraID, cameraName string) {
	message := &Message{
		Type:       "camera_removed",
		CameraID:   cameraID,
		CameraName: cameraName,
	}
	s.hub.Broadcast(message)
	log.Printf("Broadcasted camera removed: %s", cameraName)
}

// NotifyCameraUpdated отправляет уведомление об обновлении камеры
func (s *NotificationService) NotifyCameraUpdated(camera interface{}) {
	message := &Message{
		Type:   "camera_updated",
		Camera: camera,
	}
	s.hub.Broadcast(message)
	log.Printf("Broadcasted camera updated")
}

// NotifyStatsUpdate отправляет обновленную статистику системы
func (s *NotificationService) NotifyStatsUpdate(stats interface{}) {
	message := &Message{
		Type: "stats_update",
		Data: stats,
	}
	s.hub.Broadcast(message)
	log.Printf("Broadcasted stats update")
}

// StartHeartbeat запускает периодическую отправку статистики
func (s *NotificationService) StartHeartbeat(ctx context.Context, statsProvider func() interface{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("WebSocket heartbeat started")

	for {
		select {
		case <-ctx.Done():
			log.Println("WebSocket heartbeat stopped")
			return
		case <-ticker.C:
			if s.hub.GetClientCount() > 0 {
				stats := statsProvider()
				s.NotifyStatsUpdate(stats)
			}
		}
	}
}

// GetConnectedClients возвращает количество подключенных клиентов
func (s *NotificationService) GetConnectedClients() int {
	return s.hub.GetClientCount()
}
