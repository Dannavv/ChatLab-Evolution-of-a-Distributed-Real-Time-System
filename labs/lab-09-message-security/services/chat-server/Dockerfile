FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o main labs/lab-09-message-security/services/chat-server/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/labs/lab-09-message-security/services/chat-server/static static
COPY --from=builder /app/shared shared
EXPOSE 8080
CMD ["./main"]
