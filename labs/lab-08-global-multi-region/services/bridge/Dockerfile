FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o bridge labs/lab-08-global-multi-region/services/bridge/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/bridge .
EXPOSE 8082
CMD ["./bridge"]
