# Build stage
FROM node:18-alpine AS builder

# Set working directory
WORKDIR /app

# Copy package files
COPY package.json package-lock.json* .npmrc ./

# Install dependencies
RUN npm cache clean --force && npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build

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