.PHONY: build clean test install dev deps docker run help

# Переменные
APP_NAME := ocuai
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)

# Go настройки
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
CGO_ENABLED := 1

# Директории
BUILD_DIR := build
WEB_DIR := web
DIST_DIR := $(WEB_DIR)/dist

help: ## Показать справку
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

deps: ## Установить зависимости
	@echo "Установка Go зависимостей..."
	go mod download
	go mod tidy
	@echo "Проверка наличия OpenCV..."
	@pkg-config --exists opencv4 || (echo "Ошибка: OpenCV не найден. Установите: sudo apt-get install libopencv-dev" && exit 1)
	@echo "Установка веб-зависимостей..."
	cd $(WEB_DIR) && npm install

build-web: ## Собрать веб-интерфейс
	@echo "Сборка веб-интерфейса..."
	cd $(WEB_DIR) && npm run build

build-go: ## Собрать Go приложение
	@echo "Сборка Go приложения..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

build: deps build-web build-go ## Полная сборка проекта
	@echo "✅ Сборка завершена: $(BUILD_DIR)/$(APP_NAME)"
	@echo "Версия: $(VERSION)"
	@echo "Коммит: $(COMMIT)"

dev: ## Запуск в режиме разработки
	@echo "Запуск в режиме разработки..."
	go run ./cmd/$(APP_NAME)

run: build ## Собрать и запустить
	@echo "Запуск $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

test: ## Запустить тесты
	@echo "Запуск тестов..."
	go test -v ./...

clean: ## Очистить сборочные файлы
	@echo "Очистка..."
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	go clean -cache

install: build ## Установить в систему
	@echo "Установка $(APP_NAME)..."
	sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	sudo mkdir -p /etc/$(APP_NAME)
	sudo mkdir -p /var/lib/$(APP_NAME)
	sudo mkdir -p /var/log/$(APP_NAME)
	@if [ ! -f /etc/$(APP_NAME)/config.yaml ]; then \
		echo "Создание базовой конфигурации..."; \
		sudo ./$(BUILD_DIR)/$(APP_NAME) --help || true; \
	fi
	@echo "✅ $(APP_NAME) установлен в /usr/local/bin/$(APP_NAME)"

docker-build: ## Собрать Docker образ
	@echo "Сборка Docker образа..."
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .
	@echo "✅ Docker образ собран: $(APP_NAME):$(VERSION)"

docker-run: ## Запустить Docker контейнер
	@echo "Запуск Docker контейнера..."
	docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		-p 8554:8554 \
		-p 8555:8555 \
		-v $(APP_NAME)_data:/app/data \
		--restart unless-stopped \
		$(APP_NAME):latest

docker-stop: ## Остановить Docker контейнер
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

docker-logs: ## Показать логи Docker контейнера
	docker logs -f $(APP_NAME)

# Сборка для разных платформ
build-linux-amd64: ## Собрать для Linux AMD64
	@echo "Сборка для Linux AMD64..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/$(APP_NAME)

build-linux-arm64: ## Собрать для Linux ARM64
	@echo "Сборка для Linux ARM64..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./cmd/$(APP_NAME)

build-all: build-linux-amd64 build-linux-arm64 ## Собрать для всех платформ
	@echo "✅ Сборка для всех платформ завершена"

release: clean build-all ## Подготовить релиз
	@echo "Подготовка релиза $(VERSION)..."
	mkdir -p $(BUILD_DIR)/release
	cd $(BUILD_DIR) && tar -czf release/$(APP_NAME)-$(VERSION)-linux-amd64.tar.gz $(APP_NAME)-linux-amd64
	cd $(BUILD_DIR) && tar -czf release/$(APP_NAME)-$(VERSION)-linux-arm64.tar.gz $(APP_NAME)-linux-arm64
	@echo "✅ Релиз подготовлен в $(BUILD_DIR)/release/"

# Утилиты разработки
fmt: ## Форматировать код
	go fmt ./...
	cd $(WEB_DIR) && npm run format

lint: ## Проверить код линтером
	golangci-lint run
	cd $(WEB_DIR) && npm run lint

# Системные задачи
systemd-install: install ## Установить systemd сервис
	@echo "Установка systemd сервиса..."
	sudo cp scripts/$(APP_NAME).service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable $(APP_NAME)
	@echo "✅ Systemd сервис установлен. Запуск: sudo systemctl start $(APP_NAME)"

systemd-uninstall: ## Удалить systemd сервис
	@echo "Удаление systemd сервиса..."
	sudo systemctl stop $(APP_NAME) || true
	sudo systemctl disable $(APP_NAME) || true
	sudo rm -f /etc/systemd/system/$(APP_NAME).service
	sudo systemctl daemon-reload
	@echo "✅ Systemd сервис удален"

# Управление данными
backup: ## Создать резервную копию данных
	@echo "Создание резервной копии..."
	tar -czf $(APP_NAME)-backup-$(shell date +%Y%m%d-%H%M%S).tar.gz \
		-C ~/.$(APP_NAME) . 2>/dev/null || \
		tar -czf $(APP_NAME)-backup-$(shell date +%Y%m%d-%H%M%S).tar.gz \
		-C /var/lib/$(APP_NAME) . 2>/dev/null || \
		echo "Данные не найдены"

# Информация
version: ## Показать версию
	@echo "$(APP_NAME) версия $(VERSION) ($(COMMIT))"
	@echo "Дата сборки: $(BUILD_DATE)"

status: ## Показать статус системы
	@echo "Статус $(APP_NAME):"
	@systemctl is-active $(APP_NAME) 2>/dev/null || echo "Сервис не запущен"
	@curl -s http://localhost:8080/api/health | jq . 2>/dev/null || echo "API недоступно" 