# Go2RTC Auto-Start Setup ✅

## 🎯 Проблема решена!

Go2rtc теперь **автоматически запускается** вместе с OcuAI и доступен по адресу: **http://10.0.1.2:1984/**

## 🚀 Как запустить:

### Вариант 1: Упрощенный скрипт (рекомендуется)
```bash
./scripts/dev-postgres.sh
```

### Вариант 2: Ручной запуск
```bash
# 1. Запуск PostgreSQL
make -f Makefile.postgres docker-up

# 2. Сборка программы
make -f Makefile.postgres build

# 3. Запуск frontend
cd web && npm run dev &

# 4. Запуск backend с автозапуском go2rtc
POSTGRES_HOST=localhost \
POSTGRES_PORT=5432 \
POSTGRES_USER=ocuai \
POSTGRES_PASSWORD=ocuai123 \
POSTGRES_DB=ocuai \
POSTGRES_SSLMODE=disable \
PORT=8080 \
GO2RTC_PATH=./data/go2rtc/bin/go2rtc \
GO2RTC_CONFIG=./data/go2rtc/go2rtc.yaml \
./bin/ocuai-postgres
```

## 🔗 Доступные адреса:

- **Go2rtc веб-интерфейс**: http://10.0.1.2:1984/
- **Go2rtc API**: http://10.0.1.2:1984/api
- **OcuAI Backend**: http://localhost:8080/
- **OcuAI Frontend**: http://localhost:3000/

## ✅ Что исправлено:

1. **Автозапуск go2rtc** в `cmd/ocuai-postgres/main.go`
2. **Go2rtc Manager** создается и запускается после генерации конфигурации  
3. **Graceful shutdown** - go2rtc корректно останавливается при завершении
4. **Создан упрощенный dev-postgres.sh** скрипт

## 🔧 Изменения в коде:

### cmd/ocuai-postgres/main.go
```go
// Добавлен импорт
import "ocuai/internal/go2rtc"

// Добавлен автозапуск go2rtc
log.Println("Starting go2rtc streaming server...")
var go2rtcManager *go2rtc.Manager
go2rtcManager, err = go2rtc.New("./data/go2rtc")
if err != nil {
    log.Printf("Warning: Failed to create go2rtc manager: %v", err)
} else {
    if err := go2rtcManager.Start(); err != nil {
        log.Printf("Warning: Failed to start go2rtc: %v", err)
    } else {
        log.Println("✅ Go2rtc started successfully")
    }
}

// Добавлен graceful shutdown
if go2rtcManager != nil {
    log.Println("Stopping go2rtc...")
    if err := go2rtcManager.Stop(); err != nil {
        log.Printf("Warning: Failed to stop go2rtc: %v", err)
    } else {
        log.Println("✅ Go2rtc stopped successfully")
    }
}
```

## 🎊 Результат:

Go2rtc теперь работает **автоматически** - больше не нужно запускать его вручную! 