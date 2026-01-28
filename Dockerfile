FROM golang:1.24-alpine AS builder

# git нужен для работы с модулями
# RUN apk add --no-cache git

WORKDIR /app

# Копируем .env, конфиг и миграции для последующего runtime
COPY .env .env
COPY config config

COPY go.mod go.sum ./
# RUN go mod download

# Копируем весь исходник
COPY . .

COPY main .

FROM alpine:3.21.3

# Устанавливаем лишь сертификаты
# RUN apk add --no-cache ca-certificates

# Переключаемся в директорию бинаря
WORKDIR /app/cmd

# Копируем .env, конфиг, миграции
COPY --from=builder /app/.env .env
COPY --from=builder /app/config ./config

# Копируем сам бинарь
COPY --from=builder /app/main .

# Делаем исполняемым
RUN chmod +x ./main

EXPOSE 9001
ENTRYPOINT ["./main"]