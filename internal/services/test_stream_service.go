package services

// TestStreamService сервис для тестирования стримов
type TestStreamService struct {
	cameraService *CameraService
}

// NewTestStreamService создает новый сервис тестирования стримов
func NewTestStreamService(cameraService *CameraService) *TestStreamService {
	return &TestStreamService{
		cameraService: cameraService,
	}
}

// TestStream тестирует подключение к стриму (пока не реализовано)
func (s *TestStreamService) TestStream(url string) error {
	// TODO: реализовать тестирование стрима
	return nil
}
