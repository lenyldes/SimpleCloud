# Stage 1: Build binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o storage-service ./cmd/main.go

# Stage 2: Minimal runtime image
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

RUN addgroup -S app && adduser -S app -G app && \
    mkdir -p /storage && chown -R app:app /storage

WORKDIR /app

COPY --from=builder /app/storage-service .

USER app

EXPOSE 8080

CMD ["./storage-service"]
