# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy root go.mod and go.sum first for dependency resolution
COPY go.mod go.sum ./

# Copy shared module
COPY shared/ ./shared/

# Copy the service-specific files
COPY services/realtime-service/go.mod services/realtime-service/go.sum ./services/realtime-service/
COPY services/realtime-service/ ./services/realtime-service/

# Change to service directory for building
WORKDIR /app/services/realtime-service

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create app user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/services/realtime-service/server .

# Copy migrations (if any)
COPY --from=builder /app/services/realtime-service/migrations ./migrations

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8085

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8085/health || exit 1

# Run the application
CMD ["./server"]