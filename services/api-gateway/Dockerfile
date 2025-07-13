# API Gateway Dockerfile
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Create app directory
WORKDIR /app

# Copy shared dependencies first
COPY shared/ ./shared/

# Copy API gateway source
COPY services/api-gateway/ ./services/api-gateway/

# Set working directory to API gateway
WORKDIR /app/services/api-gateway

# Download dependencies
RUN go mod download
RUN go mod verify

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

# Production stage
FROM alpine:latest

# Install ca-certificates and wget for health checks
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/services/api-gateway/main .

# Copy configuration files
COPY --from=builder /app/services/api-gateway/config ./config/

# Change ownership
RUN chown -R appuser:appgroup /root/

# Switch to non-root user
USER appuser

# Expose port 8080 (API gateway default)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]