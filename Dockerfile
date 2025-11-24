# Build stage
FROM golang:1.25-alpine AS builder

# Set environment variables
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set working directory inside the container
WORKDIR /build

# Copy go.mod and go.sum files for dependency installation
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the Go binary
RUN go build -o aletheia ./cmd

# Runtime stage
FROM alpine:latest

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/aletheia .

# Copy web assets (templates and static files)
COPY --from=builder /build/web ./web

# Create uploads directory
RUN mkdir -p /app/uploads

# Expose application port
EXPOSE 1323

# Run the application
CMD ["./aletheia"]
