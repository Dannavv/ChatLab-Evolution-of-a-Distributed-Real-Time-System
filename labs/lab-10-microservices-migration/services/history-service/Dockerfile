FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o history-service labs/lab-10-microservices-migration/services/history-service/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/history-service .
COPY --from=builder /app/shared shared
EXPOSE 8082
CMD ["./history-service"]
