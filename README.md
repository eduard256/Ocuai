# Ocuai - AI Video Surveillance System

**Ocuai** - это монолитное приложение для домашнего видеонаблюдения с искусственным интеллектом, красивым веб-интерфейсом и Telegram-ботом. Система устанавливается одной командой и работает сразу после запуска.

## ⚡ Быстрый старт

### Установка через Docker
```bash
docker run -d \
  --name ocuai \
  -p 8080:8080 \
  -v ocuai_data:/app/data \
  --restart unless-stopped \
  ocuai/ocuai
```

### Установка на Linux
```bash
curl -fsSL https://raw.githubusercontent.com/your-repo/ocuai/main/scripts/install.sh | bash
```

## 🚀 Возможности

- **Zero-config** - работает сразу после установки
- **AI детекция** - распознавание людей, машин, животных через YOLOv8
- **Детекция движения** - встроенная система обнаружения движения
- **Telegram уведомления** - автоматическая отправка видео и алертов
- **Красивый веб-интерфейс** - современный UI на Svelte + Tailwind
- **RTSP поддержка** - встроенный go2rtc для работы с IP-камерами
- **Мульти-камеры** - поддержка множественных камер
- **Умная буферизация** - сохранение видео до и после события
- **ARM поддержка** - работает на Raspberry Pi

## 🏗️ Архитектура

- **Backend**: Go 1.21+ с chi router
- **Frontend**: Svelte + Tailwind CSS (встроен в бинарь)
- **AI**: YOLOv8 ONNX модель
- **Streaming**: встроенный go2rtc
- **Database**: SQLite
- **Notifications**: Telegram Bot API

## 📱 Использование

1. Откройте веб-интерфейс: `http://localhost:8080`
2. Добавьте Telegram-бот (токен и user_id)
3. Подключите IP-камеру (RTSP URL)
4. Настройте детекцию движения и AI
5. Получайте уведомления в Telegram!

## ⚙️ Конфигурация

Все настройки хранятся в `data/config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

telegram:
  token: "YOUR_BOT_TOKEN"
  allowed_users: [123456789]
  notifications_hours: "08:00-22:00"

cameras:
  - name: "Front Door"
    rtsp_url: "rtsp://192.168.1.100/stream"
    motion_detection: true
    ai_detection: true
    sensitivity: 0.7

storage:
  retention_days: 7
  max_video_size_mb: 50
```

## 🐳 Docker Compose

```yaml
version: '3.8'
services:
  ocuai:
    image: ocuai/ocuai
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - OCUAI_PORT=8080
    restart: unless-stopped
```

## 📊 Системные требования

- **Минимум**: 1GB RAM, 2GB диск
- **Рекомендуется**: 2GB RAM, 10GB диск
- **OS**: Linux (Ubuntu/Debian/CentOS), ARM64
- **Зависимости**: Docker или системные библиотеки OpenCV

## 🔒 Безопасность

- Все настройки локальные (SQLite/YAML)
- Telegram-бот работает только с авторизованными пользователями
- Видео не покидает ваш сервер
- HTTPS поддержка из коробки

## 🤝 Вклад в проект

**Важно**: Этот проект разработан как монолитное решение. Форки не поддерживаются. Все изменения принимаются только через Pull Request в основной репозиторий.

## 📄 Лицензия

MIT License. См. [LICENSE](LICENSE) для деталей.

## 🛠️ Разработка

```bash
# Клонирование
git clone https://github.com/your-repo/ocuai.git
cd ocuai

# Установка зависимостей
go mod download

# Сборка фронтенда
cd web && npm install && npm run build

# Запуск
go run cmd/ocuai/main.go
```

---

**Ocuai** © 2024. Создано с ❤️ для домашней безопасности.