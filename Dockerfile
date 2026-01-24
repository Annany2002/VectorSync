# Multi stage build

# Builder stage
FROM golang:1.24-alpine AS builder

# Working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.* ./

# Download go dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o vectorsync ./cmd/vector-sync/main.go

# Final stage
FROM alpine:latest

# Install necessary runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user
RUN adduser -D -u 10001 vectorsync

# Working directory
WORKDIR /app

# Copy the built binary from builder stage
COPY --from=builder /app/vectorsync .

# Create logs directory and change ownership
RUN mkdir logs && chown -R vectorsync:vectorsync /app

# Switch to non-root user
USER vectorsync

# Expose the port
EXPOSE 6309

# Run the application
CMD ["./vectorsync"]
