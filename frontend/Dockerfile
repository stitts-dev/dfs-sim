# Build stage
FROM node:18-alpine AS builder

# Set working directory
WORKDIR /app

# Accept build arguments
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL

# Debug: Print the environment variable
RUN echo "VITE_API_URL during build: $VITE_API_URL"

# Copy package files
COPY package.json package-lock.json* .npmrc ./

# Install dependencies
RUN npm cache clean --force && npm ci

# Copy source code
COPY . .

# Debug: Print environment variables just before build
RUN env | grep VITE || echo "No VITE env vars found"

# Build the application with environment variables explicitly passed
RUN VITE_API_URL="$VITE_API_URL" npm run build

# Production stage
FROM node:18-alpine

WORKDIR /app

# Copy built application
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package.json ./

# Install serve to run the application
RUN npm install -g serve

# Expose port
EXPOSE 5173

# Run the application
CMD ["serve", "-s", "dist", "-l", "5173"]