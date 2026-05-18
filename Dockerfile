FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY .env .env
COPY config config

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /app/main ./internal/cmd/main.go

FROM alpine:3.21.3

WORKDIR /app/cmd

COPY --from=builder /app/.env .env
COPY --from=builder /app/config ./config
COPY --from=builder /app/main .

EXPOSE 3003
ENTRYPOINT ["./main"]
