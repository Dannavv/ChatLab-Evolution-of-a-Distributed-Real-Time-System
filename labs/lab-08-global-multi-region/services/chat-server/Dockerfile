FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o server labs/lab-08-global-multi-region/services/chat-server/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/labs/lab-08-global-multi-region/services/chat-server/static static
COPY --from=builder /app/shared shared
EXPOSE 8080
CMD ["./server"]
