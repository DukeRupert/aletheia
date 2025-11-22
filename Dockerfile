# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary, -ldflags for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o aletheia ./cmd

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/aletheia .

# Copy migrations (needed for runtime)
COPY --from=builder /build/internal/migrations ./internal/migrations

# Copy web assets (templates and static files)
COPY --from=builder /build/web ./web

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app && \
    chown -R app:app /app

# Create uploads directory
RUN mkdir -p /app/uploads && chown -R app:app /app/uploads

USER app

# Expose application port
EXPOSE 1323

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:1323/health || exit 1

# Run the application
CMD ["./aletheia"]
