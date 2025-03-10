FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o qps-counter ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/qps-counter .
COPY config/config.yaml /etc/qps-counter/
EXPOSE 8080
CMD ["./qps-counter", "-config=/etc/qps-counter/config.yaml"]