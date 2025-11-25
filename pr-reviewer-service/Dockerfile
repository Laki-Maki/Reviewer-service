# Этап сборки
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем модули и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код и собираем бинарь
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o pr-reviewer ./cmd/app

# Финальный образ
FROM alpine:latest

WORKDIR /app

# Устанавливаем migrate
RUN apk add --no-cache curl && \
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/

COPY --from=builder /app/pr-reviewer .

EXPOSE 8080

# Запуск приложения
CMD ["./pr-reviewer"]
