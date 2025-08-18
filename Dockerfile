# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker's build cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 disables cgo for a statically linked binary
# -o specifies the output binary name and path
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/clash-subscription-updater .

# Final stage
FROM alpine:latest

WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/clash-subscription-updater .

# require mount ./clash-subscription-updater.yaml:/root/clash-subscription-updater.yaml
# require mount xxx/clash/config.yaml:/root/config.yaml

# Command to run the application
CMD ["./clash-subscription-updater"]