#!/bin/bash

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Starting DFS Lineup Optimizer locally..."
echo "========================================"

# Check if PostgreSQL is running locally
echo -e "\n${YELLOW}Checking PostgreSQL...${NC}"
if pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PostgreSQL is running${NC}"
else
    echo -e "${RED}✗ PostgreSQL is not running${NC}"
    echo "Please start PostgreSQL on port 5432"
    echo "If using Homebrew: brew services start postgresql"
    exit 1
fi

# Check if Redis is running locally
echo -e "\n${YELLOW}Checking Redis...${NC}"
if redis-cli ping > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Redis is running${NC}"
else
    echo -e "${RED}✗ Redis is not running${NC}"
    echo "Please start Redis on port 6379"
    echo "If using Homebrew: brew services start redis"
    exit 1
fi

# Create database if it doesn't exist
echo -e "\n${YELLOW}Setting up PostgreSQL database...${NC}"
createdb -h localhost -p 5432 -U postgres dfs_optimizer 2>/dev/null || echo "Database already exists"

# Start backend
echo -e "\n${YELLOW}Starting Backend...${NC}"
cd backend

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "Creating backend .env file..."
    cat > .env << EOF
PORT=8080
ENV=development
DATABASE_URL=postgres://postgres:postgres@localhost:5432/dfs_optimizer?sslmode=disable
REDIS_URL=redis://localhost:6379/0
JWT_SECRET=your-dev-secret-key
CORS_ORIGINS=http://localhost:5173
MAX_LINEUPS=150
OPTIMIZATION_TIMEOUT=30
MAX_SIMULATIONS=10000
SIMULATION_WORKERS=4
EOF
fi

# Install Go dependencies
echo "Installing Go dependencies..."
go mod download

# Run migrations
echo "Running database migrations..."
go run cmd/migrate/main.go up

# Seed sample data
echo "Seeding sample data..."
go run cmd/migrate/main.go seed || true

# Start backend server in background
echo -e "${GREEN}Starting backend server on http://localhost:8080${NC}"
go run cmd/server/main.go &
BACKEND_PID=$!
echo "Backend PID: $BACKEND_PID"

# Wait for backend to start
sleep 3

# Start frontend
echo -e "\n${YELLOW}Starting Frontend...${NC}"
cd ../frontend

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "Creating frontend .env file..."
    cat > .env << EOF
VITE_API_URL=http://localhost:8080/api/v1
EOF
fi

# Install npm dependencies
echo "Installing npm dependencies..."
npm install

# Start frontend dev server
echo -e "${GREEN}Starting frontend server on http://localhost:5173${NC}"
npm run dev &
FRONTEND_PID=$!
echo "Frontend PID: $FRONTEND_PID"

# Create stop script
cat > ../stop-local.sh << EOF
#!/bin/bash
echo "Stopping services..."
kill $BACKEND_PID 2>/dev/null && echo "Backend stopped"
kill $FRONTEND_PID 2>/dev/null && echo "Frontend stopped"
echo "Services stopped"
EOF
chmod +x ../stop-local.sh

echo -e "\n${GREEN}========================================"
echo "✓ All services started successfully!"
echo "========================================"
echo ""
echo "Backend API: http://localhost:8080"
echo "Frontend UI: http://localhost:5173"
echo ""
echo "To stop all services, run: ./stop-local.sh"
echo "========================================"
echo ""
echo "Press Ctrl+C to stop all services"

# Wait for interrupt
wait