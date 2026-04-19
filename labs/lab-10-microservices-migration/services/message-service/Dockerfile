FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o message-service labs/lab-10-microservices-migration/services/message-service/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/message-service .
COPY --from=builder /app/shared shared
EXPOSE 8081
CMD ["./message-service"]
