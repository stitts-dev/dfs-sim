# AI Recommendations Service Dockerfile
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Create app directory
WORKDIR /app

# Copy shared dependencies first
COPY shared/ ./shared/

# Copy ai-recommendations service source
COPY services/ai-recommendations-service/ ./services/ai-recommendations-service/

# Set working directory to ai-recommendations service
WORKDIR /app/services/ai-recommendations-service

# Download dependencies
RUN go mod download
RUN go mod verify

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

# Production stage
FROM alpine:latest

# Install ca-certificates for HTTPS calls (required for Claude API)
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/services/ai-recommendations-service/main .

# Change ownership
RUN chown -R appuser:appgroup /root/

# Switch to non-root user
USER appuser

# Expose port 8084 (ai-recommendations service default)
EXPOSE 8084

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8084/health || exit 1

# Run the application
CMD ["./main"]