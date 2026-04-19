FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o api labs/lab-06-chaos-and-resilience/services/chat-api/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/api .
COPY --from=builder /app/labs/lab-06-chaos-and-resilience/services/chat-api/static static
COPY --from=builder /app/shared shared
EXPOSE 8080
CMD ["./api"]
