# Этап 1: Сборка фронтенда
FROM node:18-alpine AS web-builder

WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci --only=production

COPY web/ ./
RUN npm run build

# Этап 2: Сборка Go приложения
FROM golang:1.21-alpine AS go-builder

# Установка системных зависимостей для OpenCV
RUN apk add --no-cache \
    git \
    gcc \
    g++ \
    musl-dev \
    pkgconfig \
    opencv-dev

WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Копируем собранный фронтенд
COPY --from=web-builder /app/web/dist ./web/dist

# Сборка приложения
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE

RUN CGO_ENABLED=1 go build \
    -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE} -s -w" \
    -o ocuai \
    ./cmd/ocuai

# Этап 3: Финальный образ
FROM alpine:latest

# Установка runtime зависимостей
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    opencv \
    ffmpeg \
    curl

# Создание пользователя
RUN addgroup -g 1001 -S ocuai && \
    adduser -u 1001 -S ocuai -G ocuai

# Создание директорий
RUN mkdir -p /app/data /app/models /app/videos && \
    chown -R ocuai:ocuai /app

WORKDIR /app

# Копирование приложения
COPY --from=go-builder /app/ocuai .
COPY --chown=ocuai:ocuai scripts/docker-entrypoint.sh .

# Права на выполнение
RUN chmod +x ocuai docker-entrypoint.sh

# Переключение на пользователя
USER ocuai

# Переменные окружения
ENV OCUAI_DATA_DIR=/app/data \
    OCUAI_HOST=0.0.0.0 \
    OCUAI_PORT=8080 \
    OCUAI_DATABASE_PATH=/app/data/ocuai.db \
    OCUAI_VIDEO_PATH=/app/data/videos

# Тома
VOLUME ["/app/data", "/app/models"]

# Порты
EXPOSE 8080 8554 8555

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Точка входа
ENTRYPOINT ["./docker-entrypoint.sh"]
CMD ["./ocuai"] 