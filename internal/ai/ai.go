package ai

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"sync"

	"ocuai/internal/config"

	"gocv.io/x/gocv"
)

// Detection представляет результат детекции
type Detection struct {
	Class      string  `json:"class"`
	Confidence float32 `json:"confidence"`
	BBox       BBox    `json:"bbox"`
}

// BBox представляет ограничивающий прямоугольник
type BBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Processor обрабатывает видео с помощью AI
type Processor struct {
	config      config.AIConfig
	net         *gocv.Net
	outputNames []string
	classes     []string
	enabled     bool
	mu          sync.RWMutex
}

// New создает новый AI процессор
func New(cfg config.AIConfig) (*Processor, error) {
	processor := &Processor{
		config:  cfg,
		classes: cfg.Classes,
		enabled: cfg.Enabled,
	}

	if cfg.Enabled {
		if err := processor.loadModel(); err != nil {
			log.Printf("Failed to load AI model, running without AI: %v", err)
			processor.enabled = false
		}
	}

	return processor, nil
}

// loadModel загружает ONNX модель
func (p *Processor) loadModel() error {
	if _, err := os.Stat(p.config.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", p.config.ModelPath)
	}

	// Загружаем модель
	net := gocv.ReadNet(p.config.ModelPath, "")
	if net.Empty() {
		return fmt.Errorf("failed to load model from %s", p.config.ModelPath)
	}

	// Устанавливаем backend
	switch p.config.DeviceType {
	case "gpu":
		net.SetPreferableBackend(gocv.NetBackendCUDA)
		net.SetPreferableTarget(gocv.NetTargetCUDA)
	default:
		net.SetPreferableBackend(gocv.NetBackendOpenCV)
		net.SetPreferableTarget(gocv.NetTargetCPU)
	}

	// Получаем имена выходных слоев
	outputNames := net.GetUnconnectedOutLayersNames()

	p.net = &net
	p.outputNames = outputNames

	log.Printf("AI model loaded successfully: %s", p.config.ModelPath)
	return nil
}

// Close закрывает процессор
func (p *Processor) Close() {
	if p.net != nil {
		p.net.Close()
	}
}

// IsEnabled возвращает статус AI
func (p *Processor) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// SetEnabled включает/выключает AI
func (p *Processor) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if enabled && !p.enabled && p.net == nil {
		if err := p.loadModel(); err != nil {
			log.Printf("Failed to enable AI: %v", err)
			return
		}
	}

	p.enabled = enabled
	log.Printf("AI processing %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

// ProcessFrame обрабатывает кадр и возвращает детекции
func (p *Processor) ProcessFrame(frame gocv.Mat) ([]Detection, error) {
	if !p.IsEnabled() || p.net == nil {
		return nil, nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Подготавливаем blob из изображения
	blob := gocv.BlobFromImage(frame, 1.0/255.0, image.Pt(640, 640), gocv.NewScalar(0, 0, 0, 0), true, false, gocv.MatTypeCV32F)
	defer blob.Close()

	// Устанавливаем входные данные
	p.net.SetInput(blob, "")

	// Выполняем forward pass
	outputs := p.net.ForwardLayers(p.outputNames)
	defer func() {
		for _, output := range outputs {
			output.Close()
		}
	}()

	if len(outputs) == 0 {
		return nil, fmt.Errorf("no outputs from model")
	}

	// Парсим результаты
	detections := p.parseDetections(outputs[0], frame.Cols(), frame.Rows())

	// Фильтруем по confidence
	var filteredDetections []Detection
	for _, det := range detections {
		if det.Confidence >= p.config.Threshold {
			filteredDetections = append(filteredDetections, det)
		}
	}

	return filteredDetections, nil
}

// parseDetections парсит выходные данные модели
func (p *Processor) parseDetections(output gocv.Mat, frameWidth, frameHeight int) []Detection {
	var detections []Detection

	// YOLOv8 output format: [batch_size, num_classes + 4, num_detections]
	// где 4 - это x, y, width, height

	rows := output.Size()[1] // num_detections
	cols := output.Size()[2] // num_classes + 4

	if cols < 4 {
		return detections
	}

	for i := 0; i < rows; i++ {
		// Получаем координаты bounding box
		x := output.GetFloatAt(0, i, 0)
		y := output.GetFloatAt(0, i, 1)
		w := output.GetFloatAt(0, i, 2)
		h := output.GetFloatAt(0, i, 3)

		// Находим класс с максимальной вероятностью
		maxConfidence := float32(0)
		classID := -1

		for j := 4; j < cols; j++ {
			confidence := output.GetFloatAt(0, i, j)
			if confidence > maxConfidence {
				maxConfidence = confidence
				classID = j - 4
			}
		}

		// Проверяем threshold и валидность класса
		if maxConfidence < p.config.Threshold || classID < 0 || classID >= len(p.classes) {
			continue
		}

		// Преобразуем координаты относительно размера кадра
		bbox := BBox{
			X:      int((x - w/2) * float32(frameWidth)),
			Y:      int((y - h/2) * float32(frameHeight)),
			Width:  int(w * float32(frameWidth)),
			Height: int(h * float32(frameHeight)),
		}

		// Ограничиваем координаты размерами кадра
		if bbox.X < 0 {
			bbox.X = 0
		}
		if bbox.Y < 0 {
			bbox.Y = 0
		}
		if bbox.X+bbox.Width > frameWidth {
			bbox.Width = frameWidth - bbox.X
		}
		if bbox.Y+bbox.Height > frameHeight {
			bbox.Height = frameHeight - bbox.Y
		}

		detection := Detection{
			Class:      p.classes[classID],
			Confidence: maxConfidence,
			BBox:       bbox,
		}

		detections = append(detections, detection)
	}

	return detections
}

// DetectMotion обнаруживает движение между кадрами
func DetectMotion(prevFrame, currentFrame gocv.Mat, threshold float64) bool {
	if prevFrame.Empty() || currentFrame.Empty() {
		return false
	}

	// Конвертируем в grayscale
	var gray1, gray2 gocv.Mat
	defer gray1.Close()
	defer gray2.Close()

	if prevFrame.Channels() > 1 {
		gray1 = gocv.NewMat()
		gocv.CvtColor(prevFrame, &gray1, gocv.ColorBGRToGray)
	} else {
		gray1 = prevFrame.Clone()
	}

	if currentFrame.Channels() > 1 {
		gray2 = gocv.NewMat()
		gocv.CvtColor(currentFrame, &gray2, gocv.ColorBGRToGray)
	} else {
		gray2 = currentFrame.Clone()
	}

	// Вычисляем разность
	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(gray1, gray2, &diff)

	// Применяем пороговое значение
	thresh := gocv.NewMat()
	defer thresh.Close()
	gocv.Threshold(diff, &thresh, threshold, 255, gocv.ThresholdBinary)

	// Подсчитываем количество ненулевых пикселей
	nonZero := gocv.CountNonZero(thresh)

	// Если изменилось более 1% пикселей, считаем что есть движение
	totalPixels := thresh.Rows() * thresh.Cols()
	motionPercent := float64(nonZero) / float64(totalPixels)

	return motionPercent > 0.01
}

// DrawDetections рисует детекции на кадре
func DrawDetections(frame *gocv.Mat, detections []Detection) {
	for _, det := range detections {
		// Рисуем прямоугольник
		color := image.RGBA{R: 0, G: 255, B: 0, A: 255} // Зеленый цвет
		pt1 := image.Pt(det.BBox.X, det.BBox.Y)
		pt2 := image.Pt(det.BBox.X+det.BBox.Width, det.BBox.Y+det.BBox.Height)
		gocv.Rectangle(frame, pt1, pt2, color, 2)

		// Добавляем текст с классом и confidence
		text := fmt.Sprintf("%s: %.2f", det.Class, det.Confidence)
		textSize := gocv.GetTextSize(text, gocv.FontHersheySimplex, 0.5, 1)

		// Рисуем фон для текста
		textBg := image.Rect(det.BBox.X, det.BBox.Y-textSize.Y-5, det.BBox.X+textSize.X+5, det.BBox.Y)
		gocv.Rectangle(frame, textBg.Min, textBg.Max, color, -1)

		// Рисуем текст
		textPt := image.Pt(det.BBox.X+2, det.BBox.Y-5)
		gocv.PutText(frame, text, textPt, gocv.FontHersheySimplex, 0.5, image.RGBA{R: 0, G: 0, B: 0, A: 255}, 1)
	}
}

// DownloadModel скачивает модель YOLOv8 если она не существует
func DownloadModel(modelPath string) error {
	if _, err := os.Stat(modelPath); err == nil {
		return nil // Модель уже существует
	}

	// Создаем директорию для модели
	if err := os.MkdirAll(filepath.Dir(modelPath), 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	log.Printf("Model not found at %s. Please download YOLOv8 ONNX model manually.", modelPath)
	log.Printf("You can download it from: https://github.com/ultralytics/assets/releases/download/v0.0.0/yolov8n.onnx")

	return fmt.Errorf("model not found, please download manually")
}
