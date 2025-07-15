# Sports Data Service Dockerfile
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for private repos and HTTPS)
RUN apk add --no-cache git ca-certificates

# Create app directory
WORKDIR /app

# Copy shared dependencies first
COPY shared/ ./shared/

# Copy sports data service source
COPY services/sports-data-service/ ./services/sports-data-service/

# Set working directory to sports data service
WORKDIR /app/services/sports-data-service

# Download dependencies
RUN go mod download
RUN go mod verify

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Production stage
FROM alpine:latest

# Install ca-certificates and wget for HTTPS calls and health checks
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/services/sports-data-service/server .

# Copy migrations
COPY --from=builder /app/services/sports-data-service/migrations ./migrations/

# Change ownership
RUN chown -R appuser:appgroup /root/

# Switch to non-root user
USER appuser

# Expose port 8081 (sports data service default)
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Run the application
CMD ["./server"]