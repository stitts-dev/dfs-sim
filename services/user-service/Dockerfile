# User Service Dockerfile
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for private repos and HTTPS)
RUN apk add --no-cache git ca-certificates

# Create app directory
WORKDIR /app

# Copy shared dependencies first
COPY shared/ ./shared/

# Copy user service source
COPY services/user-service/ ./services/user-service/

# Set working directory to user service
WORKDIR /app/services/user-service

# Download dependencies
RUN go mod download
RUN go mod verify

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

# Production stage
FROM alpine:latest

# Install ca-certificates for HTTPS calls to external APIs
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/services/user-service/main .

# Copy migrations if they exist
COPY --from=builder /app/services/user-service/migrations ./migrations/

# Change ownership
RUN chown -R appuser:appgroup /root/

# Switch to non-root user
USER appuser

# Expose port 8083 (user service default)
EXPOSE 8083

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1

# Run the application
CMD ["./main"]