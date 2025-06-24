#!/bin/sh

set -e

# Инициализация директорий
mkdir -p /app/data/videos /app/data/logs /app/models

# Проверка переменных окружения
echo "🚀 Запуск Ocuai..."
echo "📁 Данные: ${OCUAI_DATA_DIR:-/app/data}"
echo "🌐 Хост: ${OCUAI_HOST:-0.0.0.0}"
echo "🔌 Порт: ${OCUAI_PORT:-8080}"

# Проверка модели AI
MODEL_PATH="/app/models/yolov8n.onnx"
if [ ! -f "$MODEL_PATH" ]; then
    echo "⚠️  Модель YOLOv8 не найдена в $MODEL_PATH"
    echo "🔗 Скачайте модель: https://github.com/ultralytics/assets/releases/download/v0.0.0/yolov8n.onnx"
    echo "💡 Монтируйте модели в /app/models или установите OCUAI_AI_ENABLED=false"
fi

# Создание базовой конфигурации если не существует
CONFIG_FILE="${OCUAI_DATA_DIR:-/app/data}/config.yaml"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "📝 Создание базовой конфигурации..."
    cat > "$CONFIG_FILE" << EOF
server:
  host: "${OCUAI_HOST:-0.0.0.0}"
  port: "${OCUAI_PORT:-8080}"

storage:
  database_path: "${OCUAI_DATABASE_PATH:-/app/data/ocuai.db}"
  video_path: "${OCUAI_VIDEO_PATH:-/app/data/videos}"
  retention_days: 7
  max_video_size_mb: 50

telegram:
  token: "${OCUAI_TELEGRAM_TOKEN:-}"
  allowed_users: []
  notification_hours: "08:00-22:00"

streaming:
  rtsp_port: 8554
  webrtc_port: 8555
  buffer_size_kb: 1024

ai:
  model_path: "/app/models/yolov8n.onnx"
  enabled: ${OCUAI_AI_ENABLED:-false}
  threshold: 0.5
  classes: ["person", "car", "truck", "bus", "motorcycle", "bicycle", "dog", "cat"]
  device_type: "cpu"

cameras: []
EOF
    echo "✅ Конфигурация создана: $CONFIG_FILE"
fi

# Проверка портов
if netstat -tuln 2>/dev/null | grep -q ":${OCUAI_PORT:-8080} "; then
    echo "⚠️  Порт ${OCUAI_PORT:-8080} уже используется"
fi

# Информация о запуске
echo ""
echo "🔗 Веб-интерфейс будет доступен по адресу: http://localhost:${OCUAI_PORT:-8080}"
echo "📊 API здоровья: http://localhost:${OCUAI_PORT:-8080}/api/health"
echo ""

# Запуск приложения
exec "$@" 