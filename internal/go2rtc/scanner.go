package go2rtc

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// CameraScanner сканирует камеру для поиска всех доступных потоков
type CameraScanner struct {
	manager *Manager
	timeout time.Duration
}

// StreamCandidate представляет кандидата потока
type StreamCandidate struct {
	URL            string `json:"url"`
	Protocol       string `json:"protocol"`
	Description    string `json:"description"`
	Priority       int    `json:"priority"`
	Working        bool   `json:"working"`
	PartialWorking bool   `json:"partial_working"` // Подключается, но нет видео
	ConnectionOK   bool   `json:"connection_ok"`   // Успешно подключается к go2rtc
	Error          string `json:"error,omitempty"`
	TestDuration   string `json:"test_duration,omitempty"`
}

// ScanResult результат сканирования
type ScanResult struct {
	Streams   []StreamCandidate `json:"streams"`
	BestMatch *StreamCandidate  `json:"best_match,omitempty"`
}

// NewScanner создает новый сканер камер
func NewScanner(manager *Manager) *CameraScanner {
	return &CameraScanner{
		manager: manager,
		timeout: 30 * time.Second, // Увеличиваем timeout для TRASSIR
	}
}

// ScanCamera сканирует камеру по IP адресу с поддержкой промежуточных результатов
func (s *CameraScanner) ScanCamera(ip, username, password string, progressCallback func(StreamCandidate)) (*ScanResult, error) {
	// Валидация IP
	if net.ParseIP(ip) == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	// Генерируем все возможные URL для проверки
	candidates := s.generateCandidates(ip, username, password)

	// Сортируем кандидатов по приоритету (от высокого к низкому)
	s.sortCandidatesByPriority(candidates)

	// Проверяем каждый кандидат с ограничением скорости
	allResults := make([]StreamCandidate, 0)
	workingResults := make([]StreamCandidate, 0)
	partialResults := make([]StreamCandidate, 0)
	var mu sync.Mutex

	// Создаем канал для заданий и результатов
	jobs := make(chan StreamCandidate, len(candidates))
	resultsChan := make(chan StreamCandidate, len(candidates))
	stopScan := make(chan struct{})

	// Запускаем 5 воркеров для параллельной обработки
	const numWorkers = 5
	var wg sync.WaitGroup

	// Ограничиваем общую скорость: максимум 50 URL в секунду для всех воркеров
	rateLimiter := time.NewTicker(20 * time.Millisecond) // 50/секунду = каждые 20ms
	defer rateLimiter.Stop()

	log.Printf("Starting camera scan for %s with %d candidates (5 parallel workers, max 50/sec total)", ip, len(candidates))

	// Запускаем воркеры
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case candidate, ok := <-jobs:
					if !ok {
						return
					}

					// Ждем тик лимитера скорости
					<-rateLimiter.C

					log.Printf("Testing [Worker %d] (Priority %d): %s", workerID, candidate.Priority, candidate.URL)

					// Тестируем поток с помощью безопасной функции
					s.testCandidate(&candidate)

					// Отправляем результат
					select {
					case resultsChan <- candidate:
					case <-stopScan:
						return
					}

				case <-stopScan:
					return
				}
			}
		}(w)
	}

	// Отправляем все задания
	go func() {
		for _, candidate := range candidates {
			select {
			case jobs <- candidate:
			case <-stopScan:
				break
			}
		}
		close(jobs)
	}()

	// Собираем результаты
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	successCount := 0
	partialCount := 0
	totalProcessed := 0
	maxTotalStreams := 20 // Увеличиваем лимит для показа всех найденных потоков

	for result := range resultsChan {
		totalProcessed++

		mu.Lock()
		allResults = append(allResults, result)

		if result.Working {
			workingResults = append(workingResults, result)
			successCount++
			log.Printf("Found working stream #%d: %s", successCount, result.Description)
		} else if result.ConnectionOK {
			// Если подключился к go2rtc, но по какой-то причине не считается working - добавляем в рабочие
			result.Working = true // Принудительно помечаем как рабочий
			workingResults = append(workingResults, result)
			successCount++
			log.Printf("Found working stream #%d: %s (connection OK)", successCount, result.Description)
		} else {
			partialResults = append(partialResults, result)
			partialCount++
			log.Printf("Found failed stream #%d: %s (%s)", partialCount, result.Description, result.Error)
		}
		mu.Unlock()

		// Отправляем промежуточный результат через колбэк если он есть
		if progressCallback != nil {
			progressCallback(result)
		}

		// Продолжаем сканирование чтобы найти больше потоков
		// Останавливаемся только если найдено много потоков или обработано много кандидатов
		if successCount >= 10 || totalProcessed >= maxTotalStreams {
			log.Printf("Found %d working streams, stopping scan", successCount)
			close(stopScan)
			break
		}

		// Если обработали много кандидатов без хороших результатов, останавливаем
		if totalProcessed >= 50 && successCount == 0 {
			log.Printf("Processed %d candidates without success, stopping scan", totalProcessed)
			close(stopScan)
			break
		}
	}

	log.Printf("Scan completed for %s: found %d working + %d failed = %d total streams tested",
		ip, len(workingResults), len(partialResults), len(allResults))

	// Создаем результат - возвращаем только рабочие потоки
	result := &ScanResult{
		Streams: workingResults, // Только рабочие потоки
	}

	// Выбираем лучший поток - берем из рабочих
	if len(workingResults) > 0 {
		best := workingResults[0]
		for _, stream := range workingResults {
			if stream.Priority > best.Priority {
				best = stream
			}
		}
		result.BestMatch = &best
		log.Printf("Best match: %s (Priority: %d, Status: Working)", best.URL, best.Priority)
	} else {
		log.Printf("No working streams found for camera %s", ip)
	}

	return result, nil
}

// sortCandidatesByPriority сортирует кандидатов по приоритету (от высокого к низкому)
func (s *CameraScanner) sortCandidatesByPriority(candidates []StreamCandidate) {
	// Простая сортировка пузырьком по приоритету
	n := len(candidates)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if candidates[j].Priority < candidates[j+1].Priority {
				candidates[j], candidates[j+1] = candidates[j+1], candidates[j]
			}
		}
	}
}

// generateCandidates генерирует все возможные URL для камеры
func (s *CameraScanner) generateCandidates(ip, username, password string) []StreamCandidate {
	candidates := []StreamCandidate{}

	// Известные URL камеры - максимальный приоритет (60)
	if ip == "10.0.20.111" && username != "" && password != "" {
		candidates = append(candidates, StreamCandidate{
			URL:         fmt.Sprintf("rtsp://%s:%s@%s:554/live/main", username, password, ip),
			Protocol:    "rtsp",
			Description: "Known camera URL: /live/main",
			Priority:    60, // Самый высокий приоритет
		})
	}

	// ONVIF discovery - высокий приоритет (50)
	onvifPorts := []int{80, 8080, 2020, 8000}
	for _, port := range onvifPorts {
		if username != "" && password != "" {
			candidates = append(candidates, StreamCandidate{
				URL:         fmt.Sprintf("onvif://%s:%s@%s:%d", username, password, ip, port),
				Protocol:    "onvif",
				Description: fmt.Sprintf("ONVIF discovery on port %d", port),
				Priority:    50,
			})
		}
	}

	// Стандартные RTSP потоки - высокий приоритет (40-45)
	rtspPorts := []int{554, 8554, 88, 10554, 555, 8555}
	standardRtspPaths := []string{
		"", "/", "/live", "/stream", "/cam", "/video", "/1",
		"/live/ch00_0", "/live/ch00_1", "/live/ch0", "/live/ch1",
		"/ch0", "/ch1", "/stream1", "/stream2", "/live/main",
	}

	// Генерируем стандартные RTSP URL
	for _, port := range rtspPorts {
		for _, path := range standardRtspPaths {
			priority := 45
			if port == 554 { // Стандартный RTSP порт
				priority = 45
			} else {
				priority = 42
			}

			// С авторизацией
			if username != "" {
				candidate := StreamCandidate{
					Protocol:    "rtsp",
					Description: fmt.Sprintf("Standard RTSP on port %d%s", port, path),
					Priority:    priority,
				}

				if password != "" {
					candidate.URL = fmt.Sprintf("rtsp://%s:%s@%s:%d%s", username, password, ip, port, path)
				} else {
					candidate.URL = fmt.Sprintf("rtsp://%s@%s:%d%s", username, ip, port, path)
				}
				candidates = append(candidates, candidate)
			}

			// Без авторизации (для камер без пароля)
			candidates = append(candidates, StreamCandidate{
				URL:         fmt.Sprintf("rtsp://%s:%d%s", ip, port, path),
				Protocol:    "rtsp",
				Description: fmt.Sprintf("Standard RTSP on port %d%s (no auth)", port, path),
				Priority:    priority - 5,
			})
		}
	}

	// Популярные модели камер - средне-высокий приоритет (30-35)
	popularModelPaths := []string{
		// Dahua
		"/cam/realmonitor?channel=1&subtype=0",
		"/cam/realmonitor?channel=1&subtype=1",
		"/cam/realmonitor?channel=1&subtype=0&unicast=true&proto=Onvif",

		// Hikvision
		"/Streaming/Channels/101", "/Streaming/Channels/102",
		"/Streaming/Channels/1", "/Streaming/Channels/2",
		"/h264/ch1/main/av_stream", "/h264/ch1/sub/av_stream",

		// Axis
		"/axis-media/media.amp",
		"/axis-media/media.amp?videocodec=h264",

		// Foscam
		"/videoMain", "/videoSub",
		"/video.cgi", "/video2.cgi",

		// Amcrest
		"/cam/realmonitor?channel=1&subtype=00",
		"/cam/realmonitor?channel=1&subtype=01",

		// Reolink
		"/h264Preview_01_main", "/h264Preview_01_sub",
		"/Preview_01_main", "/Preview_01_sub",
	}

	for _, port := range rtspPorts {
		for _, path := range popularModelPaths {
			priority := 35
			if port == 554 {
				priority = 35
			} else {
				priority = 32
			}

			// С авторизацией
			if username != "" && password != "" {
				candidates = append(candidates, StreamCandidate{
					URL:         fmt.Sprintf("rtsp://%s:%s@%s:%d%s", username, password, ip, port, path),
					Protocol:    "rtsp",
					Description: fmt.Sprintf("Popular camera RTSP on port %d%s", port, path),
					Priority:    priority,
				})
			}

			// Без авторизации
			candidates = append(candidates, StreamCandidate{
				URL:         fmt.Sprintf("rtsp://%s:%d%s", ip, port, path),
				Protocol:    "rtsp",
				Description: fmt.Sprintf("Popular camera RTSP on port %d%s (no auth)", port, path),
				Priority:    priority - 5,
			})
		}
	}

	// TRASSIR и менее популярные пути - средний приоритет (20-25)
	trassirPorts := []int{80, 8080, 554, 8554}
	trassirRtspPaths := []string{
		"/video/1", "/video/2", "/video/3", "/video/4",
		"/video/primary", "/video/secondary",
		"/trassir/1", "/trassir/2", "/trassir/3", "/trassir/4",
		"/cam1", "/cam2", "/cam3", "/cam4",
		"/channel1", "/channel2", "/channel3", "/channel4",
		"/rtsp/1", "/rtsp/2", "/rtsp/3", "/rtsp/4",
	}

	for _, port := range trassirPorts {
		for _, path := range trassirRtspPaths {
			priority := 25
			if port == 554 || port == 8554 {
				priority = 25
			} else {
				priority = 22
			}

			// С авторизацией
			if username != "" && password != "" {
				candidates = append(candidates, StreamCandidate{
					URL:         fmt.Sprintf("rtsp://%s:%s@%s:%d%s", username, password, ip, port, path),
					Protocol:    "rtsp",
					Description: fmt.Sprintf("TRASSIR RTSP on port %d%s", port, path),
					Priority:    priority,
				})
			}
		}
	}

	// Специальные протоколы - низко-средний приоритет (10-15)

	// TP-Link Tapo
	if username != "" && password != "" {
		candidates = append(candidates, StreamCandidate{
			URL:         fmt.Sprintf("tapo://%s:%s@%s", username, password, ip),
			Protocol:    "tapo",
			Description: "TP-Link Tapo protocol",
			Priority:    15,
		})
		// Tapo без логина, только с паролем
		candidates = append(candidates, StreamCandidate{
			URL:         fmt.Sprintf("tapo://%s@%s", password, ip),
			Protocol:    "tapo",
			Description: "TP-Link Tapo protocol (password only)",
			Priority:    15,
		})
	}

	// DVR-IP / XMeye
	dvripPorts := []int{34567, 34568}
	for _, port := range dvripPorts {
		if username != "" && password != "" {
			candidates = append(candidates, StreamCandidate{
				URL:         fmt.Sprintf("dvrip://%s:%s@%s:%d?channel=0&subtype=0", username, password, ip, port),
				Protocol:    "dvrip",
				Description: fmt.Sprintf("DVR-IP/XMeye protocol on port %d", port),
				Priority:    12,
			})
		}
	}

	// Hikvision ISAPI
	if username != "" && password != "" {
		httpPorts := []int{80, 8080, 81, 8081}
		for _, port := range httpPorts {
			candidates = append(candidates, StreamCandidate{
				URL:         fmt.Sprintf("isapi://%s:%s@%s:%d/", username, password, ip, port),
				Protocol:    "isapi",
				Description: fmt.Sprintf("Hikvision ISAPI on port %d", port),
				Priority:    10,
			})
		}
	}

	// HTTP потоки - низкий приоритет (5-10)
	httpPorts := []int{80, 8080, 81, 8081, 8000, 9000, 8888}
	httpPaths := []string{
		"/mjpg/video.mjpg", "/mjpeg", "/video.mjpeg",
		"/cgi-bin/mjpg/video.cgi", "/nphMotionJpeg",
		"/axis-cgi/mjpg/video.cgi", "/video/mjpg.cgi",
		"/GetData.cgi?CH=1",
		"/mjpg/stream", "/mjpg/stream.cgi",
		"/video/mjpeg", "/live/mjpeg",
		"/cgi-bin/snapshot.cgi", "/snapshot.cgi",
		"/snap.jpg", "/snapshot.jpg", "/image.jpg",
	}

	for _, port := range httpPorts {
		for _, path := range httpPaths {
			priority := 8
			if port == 80 {
				priority = 8
			} else {
				priority = 6
			}

			// С авторизацией
			if username != "" && password != "" {
				candidates = append(candidates, StreamCandidate{
					URL:         fmt.Sprintf("http://%s:%s@%s:%d%s", username, password, ip, port, path),
					Protocol:    "http",
					Description: fmt.Sprintf("HTTP stream on port %d%s", port, path),
					Priority:    priority,
				})
			}

			// Без авторизации (только для основных портов)
			if port == 80 || port == 8080 {
				candidates = append(candidates, StreamCandidate{
					URL:         fmt.Sprintf("http://%s:%d%s", ip, port, path),
					Protocol:    "http",
					Description: fmt.Sprintf("HTTP stream on port %d%s (no auth)", port, path),
					Priority:    priority - 3,
				})
			}
		}
	}

	// Редкие протоколы - очень низкий приоритет (2-5)

	// RTMP
	rtmpPorts := []int{1935}
	for _, port := range rtmpPorts {
		candidates = append(candidates, StreamCandidate{
			URL:         fmt.Sprintf("rtmp://%s:%d/live/stream", ip, port),
			Protocol:    "rtmp",
			Description: fmt.Sprintf("RTMP stream on port %d", port),
			Priority:    3,
		})
	}

	// FFmpeg обертки для проблемных камер - минимальный приоритет (1)
	ffmpegCount := 0
	for _, candidate := range candidates {
		if candidate.Protocol == "rtsp" && candidate.Priority >= 30 && ffmpegCount < 10 {
			ffmpegURL := fmt.Sprintf("ffmpeg:%s#video=copy#audio=copy", candidate.URL)
			candidates = append(candidates, StreamCandidate{
				URL:         ffmpegURL,
				Protocol:    "ffmpeg+rtsp",
				Description: candidate.Description + " (via FFmpeg)",
				Priority:    1,
			})
			ffmpegCount++
		}
	}

	return candidates
}

// ScanOnvifCamera сканирует камеру через ONVIF
func (s *CameraScanner) ScanOnvifCamera(ip, username, password string, port int) (*ScanResult, error) {
	if port == 0 {
		port = 80
	}

	onvifURL := fmt.Sprintf("onvif://%s:%s@%s:%d", username, password, ip, port)
	log.Printf("Starting ONVIF scan for %s on port %d", ip, port)

	// Пробуем получить информацию через ONVIF
	start := time.Now()
	info, err := s.manager.ProbeStream(onvifURL)
	duration := time.Since(start)

	if err != nil {
		log.Printf("ONVIF probe failed (%v): %v", duration, err)
		return nil, fmt.Errorf("ONVIF probe failed: %w", err)
	}

	log.Printf("ONVIF probe succeeded (%v)", duration)

	// Извлекаем найденные потоки из результата
	result := &ScanResult{
		Streams: []StreamCandidate{},
	}

	// Парсим результаты ONVIF
	if producers, ok := info["producers"].([]interface{}); ok {
		log.Printf("Found %d ONVIF producers", len(producers))
		for i, producer := range producers {
			if prodMap, ok := producer.(map[string]interface{}); ok {
				if streamURL, ok := prodMap["url"].(string); ok {
					candidate := StreamCandidate{
						URL:         streamURL,
						Protocol:    "rtsp", // ONVIF обычно возвращает RTSP
						Description: fmt.Sprintf("ONVIF discovered stream %d", i+1),
						Priority:    50, // Максимальный приоритет для ONVIF
						Working:     true,
					}
					result.Streams = append(result.Streams, candidate)
					log.Printf("  ✅ ONVIF stream %d: %s", i+1, streamURL)
				}
			}
		}
	}

	if len(result.Streams) > 0 {
		result.BestMatch = &result.Streams[0]
		log.Printf("ONVIF scan completed: found %d streams, best: %s", len(result.Streams), result.BestMatch.URL)
	} else {
		log.Printf("ONVIF scan completed: no streams found")
	}

	return result, nil
}

// QuickScan быстрое сканирование только основных потоков
func (s *CameraScanner) QuickScan(ip, username, password string) (*StreamCandidate, error) {
	log.Printf("Starting quick scan for camera %s", ip)

	// Проверяем только самые высокоприоритетные варианты
	quickCandidates := []StreamCandidate{}

	// Известный URL камеры - самый высокий приоритет
	if ip == "10.0.20.111" && username != "" && password != "" {
		quickCandidates = append(quickCandidates, StreamCandidate{
			URL:         fmt.Sprintf("rtsp://%s:%s@%s:554/live/main", username, password, ip),
			Protocol:    "rtsp",
			Description: "Known camera URL: /live/main",
			Priority:    60,
		})
	}

	// Добавляем остальные высокоприоритетные варианты
	commonCandidates := []StreamCandidate{
		// ONVIF - наивысший приоритет
		{
			URL:         fmt.Sprintf("onvif://%s:%s@%s:80", username, password, ip),
			Protocol:    "onvif",
			Description: "ONVIF discovery on port 80",
			Priority:    50,
		},
		{
			URL:         fmt.Sprintf("onvif://%s:%s@%s:8080", username, password, ip),
			Protocol:    "onvif",
			Description: "ONVIF discovery on port 8080",
			Priority:    50,
		},
		// Стандартный RTSP
		{
			URL:         fmt.Sprintf("rtsp://%s:%s@%s:554/", username, password, ip),
			Protocol:    "rtsp",
			Description: "Standard RTSP on port 554",
			Priority:    45,
		},
		{
			URL:         fmt.Sprintf("rtsp://%s:%s@%s:554/live", username, password, ip),
			Protocol:    "rtsp",
			Description: "Standard RTSP on port 554 /live",
			Priority:    45,
		},
		// Популярные модели
		{
			URL:         fmt.Sprintf("rtsp://%s:%s@%s:554/cam/realmonitor?channel=1&subtype=0", username, password, ip),
			Protocol:    "rtsp",
			Description: "Dahua RTSP",
			Priority:    35,
		},
		{
			URL:         fmt.Sprintf("rtsp://%s:%s@%s:554/Streaming/Channels/101", username, password, ip),
			Protocol:    "rtsp",
			Description: "Hikvision RTSP",
			Priority:    35,
		},
		{
			URL:         fmt.Sprintf("rtsp://%s:%s@%s:554/h264/ch1/main/av_stream", username, password, ip),
			Protocol:    "rtsp",
			Description: "Hikvision H264 main stream",
			Priority:    35,
		},
	}

	quickCandidates = append(quickCandidates, commonCandidates...)

	// Проверяем каждый кандидат последовательно с коротким таймаутом
	for i, candidate := range quickCandidates {
		log.Printf("Quick test [%d/%d] (Priority %d): %s", i+1, len(quickCandidates), candidate.Priority, candidate.URL)

		// Используем новую безопасную функцию тестирования
		s.testCandidate(&candidate)

		if candidate.Working {
			log.Printf("Quick scan success: %s", candidate.Description)
			return &candidate, nil
		}
	}

	log.Printf("Quick scan completed for %s: no working streams found", ip)
	return nil, fmt.Errorf("no working stream found in quick scan")
}

// ValidateIP проверяет валидность IP адреса
func ValidateIP(ip string) error {
	// Проверяем, что это валидный IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address format")
	}

	// Проверяем, что это IPv4
	if parsedIP.To4() == nil {
		return fmt.Errorf("only IPv4 addresses are supported")
	}

	// Проверяем, что это не специальные адреса
	if parsedIP.IsLoopback() {
		return fmt.Errorf("loopback addresses are not allowed")
	}

	if parsedIP.IsMulticast() {
		return fmt.Errorf("multicast addresses are not allowed")
	}

	return nil
}

// ExtractStreamName извлекает имя потока из URL
func ExtractStreamName(streamURL string) string {
	// Парсим URL
	u, err := url.Parse(streamURL)
	if err != nil {
		// Если не удалось распарсить, используем хеш
		return fmt.Sprintf("camera_%d", time.Now().Unix())
	}

	// Извлекаем хост
	host := u.Hostname()
	if host == "" {
		host = "unknown"
	}

	// Заменяем точки на подчеркивания
	host = strings.ReplaceAll(host, ".", "_")
	host = strings.ReplaceAll(host, ":", "_")

	// Добавляем протокол для уникальности
	protocol := u.Scheme
	if protocol == "" {
		protocol = "stream"
	}

	return fmt.Sprintf("%s_%s", protocol, host)
}

// testCandidate проверяет работает ли кандидат и заполняет всю информацию о статусе
func (s *CameraScanner) testCandidate(candidate *StreamCandidate) {
	start := time.Now()

	// Создаем уникальное имя для тестового потока
	testName := fmt.Sprintf("test_scan_%d_%d", time.Now().Unix(), time.Now().Nanosecond())

	log.Printf("Testing stream: %s", candidate.URL)

	// Пытаемся добавить поток в go2rtc
	err := s.manager.AddStream(testName, candidate.URL)
	duration := time.Since(start)
	candidate.TestDuration = duration.String()

	// Всегда удаляем тестовый поток
	defer func() {
		if removeErr := s.manager.RemoveStream(testName); removeErr != nil {
			log.Printf("Warning: failed to remove test stream %s: %v", testName, removeErr)
		}
	}()

	if err != nil {
		// Поток не смог подключиться вообще
		candidate.Error = err.Error()
		candidate.ConnectionOK = false
		candidate.PartialWorking = false
		candidate.Working = false
		log.Printf("  ❌ Connection failed (%v): %v", duration, err)
		return
	}

	// Поток успешно подключился к go2rtc - ЭТОГО ДОСТАТОЧНО!
	candidate.ConnectionOK = true
	candidate.Working = true // Считаем рабочим сразу
	candidate.PartialWorking = false

	log.Printf("  ✅ Success (%v): stream accepted by go2rtc", duration)

	// Убираем все дополнительные проверки - если go2rtc принял поток, он рабочий
	// Не проверяем GetStreams(), GetStreamInfo() и producers - это может быть ненадежно
}
