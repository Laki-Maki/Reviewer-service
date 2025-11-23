# Этап сборки
FROM golang:1.24-alpine AS build

WORKDIR /app

# Копируем модули и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod tidy

# Копируем весь код и собираем бинарь
COPY . ./
RUN go build -o pr-reviewer ./cmd/app/main.go

# Финальный образ для запуска
FROM alpine:3.18

WORKDIR /app
COPY --from=build /app/pr-reviewer .

# Запуск приложения
CMD ["./pr-reviewer"]
