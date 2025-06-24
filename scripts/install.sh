#!/bin/bash

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Конфигурация
APP_NAME="ocuai"
REPO_URL="https://github.com/your-repo/ocuai"
VERSION="latest"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/ocuai"
CONFIG_DIR="/etc/ocuai"
LOG_DIR="/var/log/ocuai"
USER="ocuai"
GROUP="ocuai"

# Функции
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "Этот скрипт должен запускаться с правами root"
        echo "Используйте: sudo $0"
        exit 1
    fi
}

detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    else
        log_error "Не удалось определить операционную систему"
        exit 1
    fi
    
    log_info "Обнаружена ОС: $OS $VER"
}

install_dependencies() {
    log_info "Установка зависимостей..."
    
    case "$OS" in
        *"Ubuntu"*|*"Debian"*)
            apt-get update
            apt-get install -y curl wget unzip libopencv-dev ffmpeg sqlite3
            ;;
        *"CentOS"*|*"Red Hat"*|*"Rocky"*|*"AlmaLinux"*)
            yum update -y
            yum install -y curl wget unzip opencv-devel ffmpeg sqlite
            ;;
        *"Arch"*)
            pacman -Syu --noconfirm
            pacman -S --noconfirm curl wget unzip opencv ffmpeg sqlite
            ;;
        *)
            log_warning "Неизвестная ОС. Попытка установки базовых пакетов..."
            ;;
    esac
    
    log_success "Зависимости установлены"
}

create_user() {
    if ! id "$USER" &>/dev/null; then
        log_info "Создание пользователя $USER..."
        useradd --system --home-dir "$DATA_DIR" --shell /bin/false "$USER"
        log_success "Пользователь $USER создан"
    else
        log_info "Пользователь $USER уже существует"
    fi
}

create_directories() {
    log_info "Создание директорий..."
    
    mkdir -p "$DATA_DIR"/{videos,models,logs}
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    
    chown -R "$USER:$GROUP" "$DATA_DIR"
    chown -R "$USER:$GROUP" "$LOG_DIR"
    
    chmod 755 "$DATA_DIR"
    chmod 755 "$CONFIG_DIR"
    chmod 755 "$LOG_DIR"
    
    log_success "Директории созданы"
}

download_binary() {
    log_info "Скачивание $APP_NAME..."
    
    # Определяем архитектуру
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            log_error "Неподдерживаемая архитектура: $ARCH"
            exit 1
            ;;
    esac
    
    # URL для скачивания (нужно адаптировать под ваш репозиторий)
    DOWNLOAD_URL="$REPO_URL/releases/latest/download/$APP_NAME-linux-$ARCH"
    
    # Временная директория
    TMP_DIR=$(mktemp -d)
    
    # Скачиваем бинарь
    if ! curl -L -o "$TMP_DIR/$APP_NAME" "$DOWNLOAD_URL"; then
        log_error "Не удалось скачать $APP_NAME"
        log_info "Попытка сборки из исходников..."
        build_from_source
        return
    fi
    
    # Устанавливаем
    chmod +x "$TMP_DIR/$APP_NAME"
    mv "$TMP_DIR/$APP_NAME" "$INSTALL_DIR/"
    
    # Очистка
    rm -rf "$TMP_DIR"
    
    log_success "$APP_NAME установлен в $INSTALL_DIR/"
}

build_from_source() {
    log_info "Сборка из исходников..."
    
    # Проверка наличия Go
    if ! command -v go &> /dev/null; then
        log_info "Установка Go..."
        case "$OS" in
            *"Ubuntu"*|*"Debian"*)
                snap install go --classic
                ;;
            *)
                log_error "Установите Go вручную: https://golang.org/dl/"
                exit 1
                ;;
        esac
    fi
    
    # Проверка наличия Node.js
    if ! command -v node &> /dev/null; then
        log_info "Установка Node.js..."
        curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
        apt-get install -y nodejs
    fi
    
    # Клонирование репозитория
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    git clone "$REPO_URL" .
    
    # Сборка
    make build
    
    # Установка
    mv build/"$APP_NAME" "$INSTALL_DIR/"
    
    # Очистка
    cd /
    rm -rf "$TMP_DIR"
    
    log_success "$APP_NAME собран и установлен"
}

download_model() {
    log_info "Скачивание AI модели..."
    
    MODEL_URL="https://github.com/ultralytics/assets/releases/download/v0.0.0/yolov8n.onnx"
    MODEL_PATH="$DATA_DIR/models/yolov8n.onnx"
    
    if [[ ! -f "$MODEL_PATH" ]]; then
        mkdir -p "$(dirname "$MODEL_PATH")"
        
        if curl -L -o "$MODEL_PATH" "$MODEL_URL"; then
            chown "$USER:$GROUP" "$MODEL_PATH"
            log_success "AI модель скачана"
        else
            log_warning "Не удалось скачать AI модель. Скачайте вручную: $MODEL_URL"
        fi
    else
        log_info "AI модель уже существует"
    fi
}

create_config() {
    log_info "Создание конфигурации..."
    
    CONFIG_FILE="$CONFIG_DIR/config.yaml"
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        cat > "$CONFIG_FILE" << EOF
server:
  host: "0.0.0.0"
  port: "8080"

storage:
  database_path: "$DATA_DIR/ocuai.db"
  video_path: "$DATA_DIR/videos"
  retention_days: 7
  max_video_size_mb: 50

telegram:
  token: ""
  allowed_users: []
  notification_hours: "08:00-22:00"

streaming:
  rtsp_port: 8554
  webrtc_port: 8555
  buffer_size_kb: 1024

ai:
  model_path: "$DATA_DIR/models/yolov8n.onnx"
  enabled: false
  threshold: 0.5
  classes: ["person", "car", "truck", "bus", "motorcycle", "bicycle", "dog", "cat"]
  device_type: "cpu"

cameras: []
EOF
        
        chown root:root "$CONFIG_FILE"
        chmod 644 "$CONFIG_FILE"
        
        log_success "Конфигурация создана: $CONFIG_FILE"
    else
        log_info "Конфигурация уже существует"
    fi
}

install_systemd_service() {
    log_info "Установка systemd сервиса..."
    
    # Создаем временный файл сервиса
    cat > /etc/systemd/system/"$APP_NAME".service << EOF
[Unit]
Description=Ocuai - AI Video Surveillance System
Documentation=$REPO_URL
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$USER
Group=$GROUP
ExecStart=$INSTALL_DIR/$APP_NAME --config $CONFIG_DIR/config.yaml
Restart=always
RestartSec=5
StartLimitInterval=0

# Переменные окружения
Environment=OCUAI_DATA_DIR=$DATA_DIR
Environment=OCUAI_HOST=0.0.0.0
Environment=OCUAI_PORT=8080

# Ограничения безопасности
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR $LOG_DIR
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

# Рабочая директория
WorkingDirectory=$DATA_DIR

# Логирование
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$APP_NAME

# Лимиты ресурсов
LimitNOFILE=65535
LimitNPROC=4096

# Таймауты
TimeoutStartSec=30
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
EOF
    
    # Перезагружаем systemd
    systemctl daemon-reload
    systemctl enable "$APP_NAME"
    
    log_success "Systemd сервис установлен"
}

show_completion_message() {
    echo
    log_success "🎉 Установка $APP_NAME завершена!"
    echo
    echo "📋 Следующие шаги:"
    echo "  1. Отредактируйте конфигурацию: $CONFIG_DIR/config.yaml"
    echo "  2. Добавьте Telegram токен и пользователей"
    echo "  3. Запустите сервис: sudo systemctl start $APP_NAME"
    echo "  4. Проверьте статус: sudo systemctl status $APP_NAME"
    echo
    echo "🌐 Веб-интерфейс: http://localhost:8080"
    echo "📊 API здоровья: http://localhost:8080/api/health"
    echo "📁 Данные: $DATA_DIR"
    echo "📝 Логи: journalctl -u $APP_NAME -f"
    echo
    echo "📚 Документация: $REPO_URL"
    echo
}

main() {
    echo "🚀 Установка Ocuai - AI Video Surveillance System"
    echo "=================================================="
    echo
    
    check_root
    detect_os
    install_dependencies
    create_user
    create_directories
    download_binary
    download_model
    create_config
    install_systemd_service
    show_completion_message
}

# Запуск установки
main "$@" 