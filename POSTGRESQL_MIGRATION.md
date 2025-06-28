# 🚀 Миграция на PostgreSQL

## Полная переработка архитектуры системы камер

### 🎯 Что изменилось

**Было:**
- Камеры хранились в `go2rtc.yaml` файле
- Ручное редактирование конфигурации
- Нет централизованного управления
- SQLite для остальных данных

**Стало:**
- **PostgreSQL** как основная база данных
- Камеры хранятся в таблице `cameras`
- **Автоматическая генерация** `go2rtc.yaml` из БД
- **Автоматический перезапуск** go2rtc при изменениях
- **REST API** для управления камерами
- **Современный UI** с React компонентами

---

## 🏗️ Архитектура системы

```mermaid
graph TD
    A[Frontend React] --> B[REST API Go]
    B --> C[PostgreSQL Database]
    B --> D[Camera Service]
    D --> E[go2rtc.yaml Generator]
    D --> F[go2rtc Process Manager]
    E --> G[go2rtc Streaming Server]
    F --> G
    C --> H[cameras table]
```

---

## 🚀 Быстрый старт

### 1. Запуск системы

```bash
# Запуск всей системы (PostgreSQL + приложение)
make -f Makefile.postgres start

# Или в режиме разработки
make -f Makefile.postgres dev
```

### 2. Проверка статуса

```bash
# Проверить статус всех сервисов
make -f Makefile.postgres status

# Посмотреть логи
make -f Makefile.postgres logs
```

### 3. Добавление тестовых данных

```bash
# Создать тестовые камеры
make -f Makefile.postgres test-data
```

---

## 📊 Структура базы данных

### Таблица `cameras`

```sql
CREATE TABLE cameras (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    url TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

**Особенности:**
- UUID как первичный ключ
- Автоматические временные метки
- Ограничения на формат URL
- Уникальность имен камер

---

## 🌐 REST API

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/cameras` | Получить список камер |
| `POST` | `/api/cameras` | Создать новую камеру |
| `GET` | `/api/cameras/{id}` | Получить камеру по ID |
| `PUT` | `/api/cameras/{id}` | Обновить камеру |
| `DELETE` | `/api/cameras/{id}` | Удалить камеру |
| `GET` | `/api/cameras/status` | Статус системы |

### Примеры использования

#### Создание камеры
```bash
curl -X POST http://localhost:8080/api/cameras \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Living Room Camera",
    "url": "rtsp://admin:password@192.168.1.100:554/stream1",
    "description": "Main living room surveillance"
  }'
```

#### Получение списка камер
```bash
curl http://localhost:8080/api/cameras
```

#### Удаление камеры
```bash
curl -X DELETE http://localhost:8080/api/cameras/{camera-id}
```

---

## ⚙️ Автоматическая генерация конфигурации

### Как это работает

1. **При изменении камеры** → Обновление в PostgreSQL
2. **Автоматическая генерация** → Новый `go2rtc.yaml`
3. **Перезапуск go2rtc** → Применение изменений
4. **Проверка статуса** → Подтверждение работы

### Пример сгенерированного `go2rtc.yaml`

```yaml
# Go2RTC Configuration
# Generated automatically from database
# DO NOT EDIT MANUALLY

api:
  listen: ":1984"

streams:
  living_room_camera: rtsp://admin:password@192.168.1.100:554/stream1
  kitchen_camera: rtsp://admin:password@192.168.1.101:554/stream1

webrtc:
  listen: ":8555"
  candidates:
    - stun:stun.l.google.com:19302

log:
  level: info
  format: text
```

---

## 🎨 Пользовательский интерфейс

### Возможности UI

- **Просмотр всех камер** в удобном формате
- **Добавление новых камер** через форму
- **Удаление камер** с подтверждением
- **Статус камер** (активная/неактивная)
- **Автообновление** списка без перезагрузки
- **Обработка ошибок** с уведомлениями

### Доступ к UI

```
http://localhost:8080/
```

---

## 🔧 Управление системой

### Команды Makefile

```bash
# Настройка окружения
make -f Makefile.postgres setup

# Запуск PostgreSQL
make -f Makefile.postgres docker-up

# Запуск миграций
make -f Makefile.postgres migrate

# Сборка приложения
make -f Makefile.postgres build

# Полная остановка
make -f Makefile.postgres stop

# Очистка всех данных
make -f Makefile.postgres clean

# Подключение к БД
make -f Makefile.postgres db-shell

# Тестирование API
make -f Makefile.postgres test
```

### Переменные окружения

```bash
# PostgreSQL настройки
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=ocuai
export POSTGRES_PASSWORD=ocuai123
export POSTGRES_DB=ocuai
export POSTGRES_SSLMODE=disable

# Приложение
export PORT=8080
export GO2RTC_PATH=./data/go2rtc/bin/go2rtc
export GO2RTC_CONFIG=./data/go2rtc/go2rtc.yaml
```

---

## 🚨 Troubleshooting

### Частые проблемы

#### PostgreSQL не запускается
```bash
# Проверить статус контейнера
docker ps -a | grep ocuai-postgres

# Просмотреть логи
docker logs ocuai-postgres

# Перезапустить
make -f Makefile.postgres docker-down
make -f Makefile.postgres docker-up
```

#### go2rtc не запускается
```bash
# Проверить файл конфигурации
cat data/go2rtc/go2rtc.yaml

# Проверить права на файлы
ls -la data/go2rtc/

# Проверить процесс
ps aux | grep go2rtc
```

#### API не отвечает
```bash
# Проверить статус приложения
make -f Makefile.postgres status

# Проверить логи
make -f Makefile.postgres logs

# Проверить подключение к БД
make -f Makefile.postgres db-shell
```

### Логи и отладка

```bash
# Логи PostgreSQL
docker logs ocuai-postgres

# Логи приложения
tail -f data/logs/ocuai.log

# Логи go2rtc
journalctl -f | grep go2rtc
```

---

## 🔒 Безопасность

### Настройки безопасности

1. **Смените пароли** по умолчанию в production
2. **Используйте SSL** для PostgreSQL в production
3. **Настройте firewall** для нужных портов
4. **Регулярно обновляйте** зависимости

### Production конфигурация

```bash
# Безопасные пароли
export POSTGRES_PASSWORD=your_secure_password_here

# SSL для PostgreSQL
export POSTGRES_SSLMODE=require

# Ограничение доступа
export POSTGRES_HOST=your_secure_db_host
```

---

## 📈 Мониторинг

### Проверка здоровья системы

```bash
# Статус всех компонентов
curl http://localhost:8080/api/cameras/status | jq

# Количество камер
curl http://localhost:8080/api/cameras | jq '.data.total'

# Статус go2rtc
curl http://localhost:1984/api/streams
```

### Метрики для мониторинга

- Количество активных камер
- Статус подключения к PostgreSQL
- Время ответа API
- Использование ресурсов go2rtc
- Ошибки в логах

---

## 🚀 Production развертывание

### Docker Compose (рекомендуется)

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: ocuai
      POSTGRES_PASSWORD: your_secure_password
      POSTGRES_DB: ocuai
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  app:
    build: .
    environment:
      POSTGRES_HOST: postgres
      POSTGRES_PASSWORD: your_secure_password
    ports:
      - "8080:8080"
    depends_on:
      - postgres

volumes:
  postgres_data:
```

### Systemd сервис

```ini
[Unit]
Description=Ocuai Camera Management System
After=network.target postgresql.service

[Service]
Type=simple
User=ocuai
WorkingDirectory=/opt/ocuai
ExecStart=/opt/ocuai/bin/ocuai-postgres
Restart=always
RestartSec=5

Environment=POSTGRES_HOST=localhost
Environment=POSTGRES_PASSWORD=your_secure_password

[Install]
WantedBy=multi-user.target
```

---

## ✅ Преимущества новой архитектуры

### Для разработчиков
- **Чистый код** с разделением ответственности
- **Тестируемость** всех компонентов
- **Расширяемость** архитектуры
- **Документированное API**

### Для пользователей
- **Удобный веб-интерфейс**
- **Мгновенные изменения** без перезагрузки
- **Надежность** работы системы
- **Масштабируемость** под любое количество камер

### Для администраторов
- **Централизованное управление**
- **Автоматизация** процессов
- **Мониторинг** состояния системы
- **Простое резервное копирование** (PostgreSQL)

---

**🎉 Поздравляем! Система полностью готова к производственному использованию.** 