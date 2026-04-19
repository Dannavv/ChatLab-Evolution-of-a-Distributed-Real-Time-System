FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build the specific lab server
RUN go build -o main labs/lab-02-persistence-layer/services/chat-server/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/labs/lab-02-persistence-layer/services/chat-server/static static
COPY --from=builder /app/shared shared
EXPOSE 8080
CMD ["./main"]
