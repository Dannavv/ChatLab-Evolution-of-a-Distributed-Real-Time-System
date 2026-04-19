FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go.mod and sum from the root
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source
COPY . .

# Build the specific lab server
RUN go build -o main labs/lab-01-monolith-baseline/services/chat-server/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/labs/lab-01-monolith-baseline/services/chat-server/static static
EXPOSE 8080
CMD ["./main"]
