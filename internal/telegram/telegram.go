package telegram

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"ocuai/internal/config"
	"ocuai/internal/events"
	"ocuai/internal/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot представляет Telegram бота
type Bot struct {
	api          *tgbotapi.BotAPI
	config       config.TelegramConfig
	eventManager *events.Manager
	stopChan     chan struct{}
	wg           sync.WaitGroup
	allowedUsers map[int64]bool
}

// New создает новый Telegram бот
func New(cfg config.TelegramConfig, eventManager *events.Manager) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("telegram token is empty")
	}

	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	// Создаем карту разрешенных пользователей для быстрого поиска
	allowedUsers := make(map[int64]bool)
	for _, userID := range cfg.AllowedUsers {
		allowedUsers[userID] = true
	}

	bot := &Bot{
		api:          api,
		config:       cfg,
		eventManager: eventManager,
		stopChan:     make(chan struct{}),
		allowedUsers: allowedUsers,
	}

	log.Printf("Telegram bot authorized: @%s", api.Self.UserName)
	return bot, nil
}

// Start запускает бота
func (b *Bot) Start() {
	// Подписываемся на события
	b.eventManager.Subscribe(events.EventTypeMotion, b.handleMotionEvent)
	b.eventManager.Subscribe(events.EventTypeAI, b.handleAIEvent)
	b.eventManager.Subscribe(events.EventTypeCameraLost, b.handleCameraLostEvent)

	// Запускаем обработку команд
	b.wg.Add(1)
	go b.handleUpdates()

	// Запускаем периодическую отправку непроцессированых событий
	b.wg.Add(1)
	go b.processUnsentEvents()

	log.Println("Telegram bot started")
}

// Stop останавливает бота
func (b *Bot) Stop() {
	close(b.stopChan)
	b.wg.Wait()
	log.Println("Telegram bot stopped")
}

// handleUpdates обрабатывает входящие сообщения
func (b *Bot) handleUpdates() {
	defer b.wg.Done()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			if update.Message != nil {
				b.handleMessage(update.Message)
			} else if update.CallbackQuery != nil {
				b.handleCallbackQuery(update.CallbackQuery)
			}
		case <-b.stopChan:
			b.api.StopReceivingUpdates()
			return
		}
	}
}

// handleMessage обрабатывает текстовые сообщения
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	userID := message.From.ID

	// Проверяем, авторизован ли пользователь
	if !b.allowedUsers[userID] {
		b.sendMessage(userID, "❌ У вас нет доступа к этому боту.")
		log.Printf("Unauthorized access attempt from user %d (@%s)", userID, message.From.UserName)
		return
	}

	// Логируем команду
	log.Printf("Command from user %d (@%s): %s", userID, message.From.UserName, message.Text)

	switch {
	case strings.HasPrefix(message.Text, "/start"):
		b.handleStartCommand(userID)
	case strings.HasPrefix(message.Text, "/status"):
		b.handleStatusCommand(userID)
	case strings.HasPrefix(message.Text, "/cameras"):
		b.handleCamerasCommand(userID)
	case strings.HasPrefix(message.Text, "/events"):
		b.handleEventsCommand(userID, message.Text)
	case strings.HasPrefix(message.Text, "/ai"):
		b.handleAICommand(userID, message.Text)
	case strings.HasPrefix(message.Text, "/help"):
		b.handleHelpCommand(userID)
	default:
		b.sendMessage(userID, "❓ Неизвестная команда. Используйте /help для списка команд.")
	}
}

// handleCallbackQuery обрабатывает нажатия inline кнопок
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	userID := query.From.ID

	if !b.allowedUsers[userID] {
		return
	}

	// Отвечаем на callback query
	callback := tgbotapi.NewCallback(query.ID, "")
	b.api.Request(callback)

	parts := strings.Split(query.Data, ":")
	if len(parts) < 2 {
		return
	}

	action := parts[0]
	param := parts[1]

	switch action {
	case "toggle_ai":
		b.handleToggleAI(userID, param == "enable")
	case "camera_details":
		b.handleCameraDetails(userID, param)
	}
}

// handleStartCommand обрабатывает команду /start
func (b *Bot) handleStartCommand(userID int64) {
	message := `🏠 *Добро пожаловать в Ocuai!*

Система видеонаблюдения с ИИ готова к работе.

Доступные команды:
/status - статус системы
/cameras - список камер
/events - последние события
/ai - управление ИИ
/help - справка`

	b.sendMessage(userID, message)
}

// handleStatusCommand обрабатывает команду /status
func (b *Bot) handleStatusCommand(userID int64) {
	stats, err := b.eventManager.GetSystemStats()
	if err != nil {
		b.sendMessage(userID, "❌ Ошибка получения статистики: "+err.Error())
		return
	}

	message := fmt.Sprintf(`📊 *Статус системы*

🎥 Камеры: %d (онлайн: %d)
📅 События сегодня: %d
📈 Всего событий: %d
🕒 Время: %s`,
		stats["cameras_total"],
		stats["cameras_online"],
		stats["events_today"],
		stats["events_total"],
		time.Now().Format("15:04:05 02.01.2006"))

	// Добавляем inline клавиатуру
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎥 Камеры", "cameras"),
			tgbotapi.NewInlineKeyboardButtonData("📋 События", "events"),
		),
	)

	msg := tgbotapi.NewMessage(userID, message)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleCamerasCommand обрабатывает команду /cameras
func (b *Bot) handleCamerasCommand(userID int64) {
	// Здесь нужно получить камеры из storage, но пока заглушка
	message := `🎥 *Камеры системы*

1. 📹 Входная дверь - 🟢 Онлайн
2. 📹 Двор - 🟢 Онлайн
3. 📹 Гараж - 🔴 Офлайн

Детекция движения: ✅
ИИ детекция: ✅`

	b.sendMessage(userID, message)
}

// handleEventsCommand обрабатывает команду /events
func (b *Bot) handleEventsCommand(userID int64, text string) {
	events, err := b.eventManager.GetRecentEvents(5)
	if err != nil {
		b.sendMessage(userID, "❌ Ошибка получения событий: "+err.Error())
		return
	}

	if len(events) == 0 {
		b.sendMessage(userID, "📋 *Последние события*\n\nСобытий пока нет.")
		return
	}

	message := "📋 *Последние события*\n\n"
	for _, event := range events {
		icon := "📱"
		switch event.Type {
		case "motion":
			icon = "🏃"
		case "ai_detection":
			icon = "🤖"
		case "camera_lost":
			icon = "📵"
		}

		message += fmt.Sprintf("%s *%s*\n📹 %s\n🕒 %s\n\n",
			icon,
			event.Description,
			event.CameraName,
			event.CreatedAt.Format("15:04 02.01.2006"))
	}

	b.sendMessage(userID, message)
}

// handleAICommand обрабатывает команды управления ИИ
func (b *Bot) handleAICommand(userID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		message := `🤖 *Управление ИИ*

/ai status - статус ИИ
/ai enable - включить ИИ
/ai disable - выключить ИИ`
		b.sendMessage(userID, message)
		return
	}

	command := parts[1]
	switch command {
	case "status":
		// Здесь нужно получить статус AI из процессора
		message := `🤖 *Статус ИИ*

Состояние: ✅ Включен
Модель: YOLOv8n
Порог детекции: 0.5
Классы: person, car, dog, cat`
		b.sendMessage(userID, message)

	case "enable":
		// Здесь нужно включить AI
		b.sendMessage(userID, "✅ ИИ детекция включена")

	case "disable":
		// Здесь нужно выключить AI
		b.sendMessage(userID, "❌ ИИ детекция выключена")
	}
}

// handleHelpCommand обрабатывает команду /help
func (b *Bot) handleHelpCommand(userID int64) {
	message := `📚 *Справка по командам*

🏠 */start* - приветствие
📊 */status* - статус системы
🎥 */cameras* - список камер
📋 */events* [N] - последние N событий
🤖 */ai* - управление ИИ
❓ */help* - эта справка

*Автоматические уведомления:*
• 🏃 Детекция движения
• 🤖 Обнаружение объектов ИИ
• 📵 Потеря связи с камерой

*Время уведомлений:* %s`

	msg := fmt.Sprintf(message, b.config.NotificationHours)
	b.sendMessage(userID, msg)
}

// handleToggleAI обрабатывает переключение ИИ
func (b *Bot) handleToggleAI(userID int64, enable bool) {
	// Здесь нужно переключить AI в процессоре
	status := map[bool]string{true: "включена", false: "выключена"}
	icon := map[bool]string{true: "✅", false: "❌"}
	b.sendMessage(userID, fmt.Sprintf("%s ИИ детекция %s", icon[enable], status[enable]))
}

// handleCameraDetails показывает детали камеры
func (b *Bot) handleCameraDetails(userID int64, cameraID string) {
	// Здесь нужно получить детали камеры
	message := fmt.Sprintf(`🎥 *Камера: %s*

Статус: 🟢 Онлайн
Последняя активность: 30 сек назад
Детекция движения: ✅
ИИ детекция: ✅`, cameraID)

	b.sendMessage(userID, message)
}

// Event handlers

// handleMotionEvent обрабатывает события движения
func (b *Bot) handleMotionEvent(event events.Event) {
	if !b.isNotificationTimeAllowed() {
		return
	}

	message := fmt.Sprintf(`🏃 *Обнаружено движение*

🎥 Камера: %s
🕒 Время: %s`,
		event.CameraName,
		event.Timestamp.Format("15:04:05 02.01.2006"))

	b.broadcastMessage(message)
}

// handleAIEvent обрабатывает события ИИ детекции
func (b *Bot) handleAIEvent(event events.Event) {
	if !b.isNotificationTimeAllowed() {
		return
	}

	confidence := ""
	if event.Confidence > 0 {
		confidence = fmt.Sprintf(" (%.0f%%)", event.Confidence*100)
	}

	message := fmt.Sprintf(`🤖 *ИИ Детекция*

📝 %s%s
🎥 Камера: %s
🕒 Время: %s`,
		event.Description,
		confidence,
		event.CameraName,
		event.Timestamp.Format("15:04:05 02.01.2006"))

	b.broadcastMessage(message)
}

// handleCameraLostEvent обрабатывает события потери камеры
func (b *Bot) handleCameraLostEvent(event events.Event) {
	message := fmt.Sprintf(`📵 *Потеря связи с камерой*

🎥 Камера: %s
🕒 Время: %s

⚠️ Проверьте подключение камеры`,
		event.CameraName,
		event.Timestamp.Format("15:04:05 02.01.2006"))

	b.broadcastMessage(message)
}

// processUnsentEvents обрабатывает неотправленные события
func (b *Bot) processUnsentEvents() {
	defer b.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			events, err := b.eventManager.GetUnprocessedEvents()
			if err != nil {
				log.Printf("Failed to get unprocessed events: %v", err)
				continue
			}

			for _, event := range events {
				// Отправляем событие
				b.sendEventNotification(storage.Event(event))

				// Помечаем как обработанное
				if err := b.eventManager.MarkEventProcessed(event.ID); err != nil {
					log.Printf("Failed to mark event as processed: %v", err)
				}
			}

		case <-b.stopChan:
			return
		}
	}
}

// sendEventNotification отправляет уведомление о событии
func (b *Bot) sendEventNotification(event storage.Event) {
	if !b.isNotificationTimeAllowed() {
		return
	}

	icon := "📱"
	switch event.Type {
	case "motion":
		icon = "🏃"
	case "ai_detection":
		icon = "🤖"
	case "camera_lost":
		icon = "📵"
	}

	message := fmt.Sprintf(`%s *%s*

🎥 Камера: %s
🕒 Время: %s`,
		icon,
		event.Description,
		event.CameraName,
		event.CreatedAt.Format("15:04:05 02.01.2006"))

	b.broadcastMessage(message)
}

// sendMessage отправляет сообщение пользователю
func (b *Bot) sendMessage(userID int64, text string) {
	msg := tgbotapi.NewMessage(userID, text)
	msg.ParseMode = "Markdown"

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send message to user %d: %v", userID, err)
	}
}

// broadcastMessage отправляет сообщение всем пользователям
func (b *Bot) broadcastMessage(text string) {
	for userID := range b.allowedUsers {
		b.sendMessage(userID, text)
	}
}

// sendVideo отправляет видео пользователю
func (b *Bot) SendVideo(userID int64, videoPath, caption string) error {
	video := tgbotapi.NewVideo(userID, tgbotapi.FilePath(videoPath))
	video.Caption = caption
	video.ParseMode = "Markdown"

	_, err := b.api.Send(video)
	if err != nil {
		return fmt.Errorf("failed to send video: %w", err)
	}

	return nil
}

// sendPhoto отправляет фото пользователю
func (b *Bot) SendPhoto(userID int64, photoPath, caption string) error {
	photo := tgbotapi.NewPhoto(userID, tgbotapi.FilePath(photoPath))
	photo.Caption = caption
	photo.ParseMode = "Markdown"

	_, err := b.api.Send(photo)
	if err != nil {
		return fmt.Errorf("failed to send photo: %w", err)
	}

	return nil
}

// isNotificationTimeAllowed проверяет, разрешено ли отправлять уведомления
func (b *Bot) isNotificationTimeAllowed() bool {
	if b.config.NotificationHours == "" {
		return true
	}

	parts := strings.Split(b.config.NotificationHours, "-")
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
