FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o gateway labs/lab-10-microservices-migration/services/gateway/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/gateway .
COPY --from=builder /app/labs/lab-10-microservices-migration/services/gateway/static static
COPY --from=builder /app/shared shared
EXPOSE 8080
CMD ["./gateway"]
