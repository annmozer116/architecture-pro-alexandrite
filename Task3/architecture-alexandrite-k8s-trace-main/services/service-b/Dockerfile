# Стейдж сборки
FROM golang:1.23.2-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum отдельно, чтобы кэшировались зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем бинарник
RUN go build -o service-b main.go

# Стейдж рантайма
FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/service-b .

EXPOSE 8080
CMD ["./service-b"]
