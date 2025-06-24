package streaming

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"ocuai/internal/ai"
	"ocuai/internal/config"
	"ocuai/internal/events"

	"gocv.io/x/gocv"
)

// Server представляет стриминг сервер
type Server struct {
	config       config.StreamingConfig
	eventManager *events.Manager
	aiProcessor  *ai.Processor
	cameras      map[string]*CameraStream
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// CameraStream представляет поток с камеры
type CameraStream struct {
	ID              string
	Name            string
	RTSPURL         string
	Status          string
	Stream          *gocv.VideoCapture
	MotionDetection bool
	AIDetection     bool
	LastFrame       gocv.Mat
	PrevFrame       gocv.Mat
	LastMotionTime  time.Time
	IsRecording     bool
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// New создает новый стриминг сервер
func New(cfg config.StreamingConfig, eventManager *events.Manager, aiProcessor *ai.Processor) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:       cfg,
		eventManager: eventManager,
		aiProcessor:  aiProcessor,
		cameras:      make(map[string]*CameraStream),
		ctx:          ctx,
		cancel:       cancel,
	}

	return server, nil
}

// Start запускает стриминг сервер
func (s *Server) Start() error {
	log.Printf("Starting streaming server on ports RTSP:%d, WebRTC:%d", s.config.RTSPPort, s.config.WebRTCPort)

	// Здесь должна быть инициализация go2rtc сервера
	// Пока просто запускаем обработку камер
	s.wg.Add(1)
	go s.processStreams()

	return nil
}

// Close закрывает стриминг сервер
func (s *Server) Close() {
	s.cancel()

	// Останавливаем все камеры
	s.mu.Lock()
	for _, camera := range s.cameras {
		camera.stop()
	}
	s.mu.Unlock()

	s.wg.Wait()
	log.Println("Streaming server stopped")
}

// AddCamera добавляет камеру
func (s *Server) AddCamera(id, rtspURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cameras[id]; exists {
		return fmt.Errorf("camera %s already exists", id)
	}

	ctx, cancel := context.WithCancel(s.ctx)
	camera := &CameraStream{
		ID:              id,
		RTSPURL:         rtspURL,
		Status:          "connecting",
		MotionDetection: true,
		AIDetection:     true,
		ctx:             ctx,
		cancel:          cancel,
	}

	s.cameras[id] = camera

	// Запускаем обработку камеры
	camera.wg.Add(1)
	go s.processCameraStream(camera)

	log.Printf("Added camera %s with RTSP URL: %s", id, rtspURL)
	return nil
}

// RemoveCamera удаляет камеру
func (s *Server) RemoveCamera(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if camera, exists := s.cameras[id]; exists {
		camera.stop()
		delete(s.cameras, id)
		log.Printf("Removed camera %s", id)
	}
}

// GetCameraStatus возвращает статус камеры
func (s *Server) GetCameraStatus(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if camera, exists := s.cameras[id]; exists {
		return camera.Status
	}

	return "not_found"
}

// UpdateCameraSettings обновляет настройки камеры
func (s *Server) UpdateCameraSettings(id string, motionDetection, aiDetection bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	camera, exists := s.cameras[id]
	if !exists {
		return fmt.Errorf("camera %s not found", id)
	}

	camera.MotionDetection = motionDetection
	camera.AIDetection = aiDetection

	log.Printf("Updated camera %s settings: motion=%v, ai=%v", id, motionDetection, aiDetection)
	return nil
}

// processStreams основной цикл обработки потоков
func (s *Server) processStreams() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkCameraHealth()
		case <-s.ctx.Done():
			return
		}
	}
}

// checkCameraHealth проверяет здоровье камер
func (s *Server) checkCameraHealth() {
	s.mu.RLock()
	cameras := make([]*CameraStream, 0, len(s.cameras))
	for _, camera := range s.cameras {
		cameras = append(cameras, camera)
	}
	s.mu.RUnlock()

	for _, camera := range cameras {
		if camera.Status == "online" && time.Since(camera.LastMotionTime) > 5*time.Minute {
			// Камера давно не показывала активность - проверяем соединение
			if camera.Stream == nil || !camera.Stream.IsOpened() {
				camera.Status = "offline"
				s.eventManager.EmitCameraLost(camera.ID, camera.Name)
			}
		}
	}
}

// processCameraStream обрабатывает поток с камеры
func (s *Server) processCameraStream(camera *CameraStream) {
	defer camera.wg.Done()

	log.Printf("Starting stream processing for camera %s", camera.ID)

	// Пытаемся подключиться к камере
	if err := s.connectToCamera(camera); err != nil {
		log.Printf("Failed to connect to camera %s: %v", camera.ID, err)
		camera.Status = "error"
		return
	}

	camera.Status = "online"
	frameCounter := 0

	for {
		select {
		case <-camera.ctx.Done():
			return
		default:
			if !s.processFrame(camera, frameCounter) {
				// Ошибка чтения кадра
				time.Sleep(1 * time.Second)
				if err := s.reconnectCamera(camera); err != nil {
					log.Printf("Failed to reconnect camera %s: %v", camera.ID, err)
					camera.Status = "error"
					return
				}
			}
			frameCounter++
		}
	}
}

// connectToCamera подключается к камере
func (s *Server) connectToCamera(camera *CameraStream) error {
	stream, err := gocv.OpenVideoCapture(camera.RTSPURL)
	if err != nil {
		return fmt.Errorf("failed to open video capture: %w", err)
	}

	if !stream.IsOpened() {
		stream.Close()
		return fmt.Errorf("camera stream is not opened")
	}

	camera.Stream = stream
	camera.LastFrame = gocv.NewMat()
	camera.PrevFrame = gocv.NewMat()

	log.Printf("Successfully connected to camera %s", camera.ID)
	return nil
}

// reconnectCamera переподключается к камере
func (s *Server) reconnectCamera(camera *CameraStream) error {
	if camera.Stream != nil {
		camera.Stream.Close()
	}

	camera.LastFrame.Close()
	camera.PrevFrame.Close()

	return s.connectToCamera(camera)
}

// processFrame обрабатывает один кадр
func (s *Server) processFrame(camera *CameraStream, frameCounter int) bool {
	if camera.Stream == nil || !camera.Stream.IsOpened() {
		return false
	}

	// Сохраняем предыдущий кадр для детекции движения
	if !camera.LastFrame.Empty() {
		camera.PrevFrame.Close()
		camera.PrevFrame = camera.LastFrame.Clone()
	}

	// Читаем новый кадр
	if !camera.Stream.Read(&camera.LastFrame) {
		return false
	}

	if camera.LastFrame.Empty() {
		return false
	}

	// Детекция движения (каждый кадр)
	if camera.MotionDetection && !camera.PrevFrame.Empty() {
		if ai.DetectMotion(camera.PrevFrame, camera.LastFrame, 30.0) {
			now := time.Now()
			// Ограничиваем частоту событий движения (не чаще раза в 5 секунд)
			if now.Sub(camera.LastMotionTime) > 5*time.Second {
				camera.LastMotionTime = now
				s.eventManager.EmitMotionDetected(camera.ID, camera.Name)
				log.Printf("Motion detected on camera %s", camera.ID)
			}
		}
	}

	// AI детекция (каждую секунду, т.е. каждые 25 кадров при 25 FPS)
	if camera.AIDetection && s.aiProcessor.IsEnabled() && frameCounter%25 == 0 {
		detections, err := s.aiProcessor.ProcessFrame(camera.LastFrame)
		if err != nil {
			log.Printf("AI processing error for camera %s: %v", camera.ID, err)
		} else if len(detections) > 0 {
			for _, detection := range detections {
				s.eventManager.EmitAIDetection(
					camera.ID,
					camera.Name,
					detection.Class,
					detection.Confidence,
					map[string]interface{}{
						"bbox": detection.BBox,
					},
				)
				log.Printf("AI detection on camera %s: %s (%.2f)", camera.ID, detection.Class, detection.Confidence)
			}
		}
	}

	return true
}

// stop останавливает камеру
func (camera *CameraStream) stop() {
	camera.cancel()
	camera.wg.Wait()

	if camera.Stream != nil {
		camera.Stream.Close()
	}

	if !camera.LastFrame.Empty() {
		camera.LastFrame.Close()
	}

	if !camera.PrevFrame.Empty() {
		camera.PrevFrame.Close()
	}
}

// GetSnapshot возвращает снапшот с камеры
func (s *Server) GetSnapshot(cameraID string) ([]byte, error) {
	s.mu.RLock()
	camera, exists := s.cameras[cameraID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("camera %s not found", cameraID)
	}

	if camera.LastFrame.Empty() {
		return nil, fmt.Errorf("no frame available from camera %s", cameraID)
	}

	// Кодируем кадр в JPEG
	buf, err := gocv.IMEncode(".jpg", camera.LastFrame)
	if err != nil {
		return nil, fmt.Errorf("failed to encode frame: %w", err)
	}
	defer buf.Close()

	return buf.GetBytes(), nil
}

// GetCameraList возвращает список активных камер
func (s *Server) GetCameraList() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cameras := make([]map[string]interface{}, 0, len(s.cameras))
	for _, camera := range s.cameras {
		cameras = append(cameras, map[string]interface{}{
			"id":               camera.ID,
			"name":             camera.Name,
			"status":           camera.Status,
			"motion_detection": camera.MotionDetection,
			"ai_detection":     camera.AIDetection,
			"last_motion":      camera.LastMotionTime,
		})
	}

	return cameras
}
