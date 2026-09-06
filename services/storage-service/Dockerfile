# Stage 1: Build binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o storage-service ./cmd/main.go

# Stage 2: Minimal runtime image
FROM alpine:3.21

RUN apk --no-cache add ca-certificates su-exec

RUN addgroup -g 1000 -S app && adduser -u 1000 -S app -G app

WORKDIR /app

COPY --from=builder /app/storage-service .
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
CMD ["./storage-service"]
