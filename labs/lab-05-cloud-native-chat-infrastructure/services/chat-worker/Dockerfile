FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o worker labs/lab-05-cloud-native-chat-infrastructure/services/chat-worker/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/worker .
COPY --from=builder /app/shared shared
EXPOSE 8081
CMD ["./worker"]
