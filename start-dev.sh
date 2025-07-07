#!/bin/bash

# Start the development servers

echo "Starting DFS Optimizer Development Environment..."

# Function to cleanup on exit
cleanup() {
    echo -e "\n\nShutting down servers..."
    kill $BACKEND_PID $FRONTEND_PID 2>/dev/null
    exit
}

# Set up trap to cleanup on script exit
trap cleanup EXIT INT TERM

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    exit 1
fi

# Check if Node is installed
if ! command -v node &> /dev/null; then
    echo "Error: Node.js is not installed. Please install Node.js first."
    exit 1
fi

# Start backend server
echo "Starting backend server on port 8080..."
cd backend
go run cmd/server/main.go &
BACKEND_PID=$!
cd ..

# Wait a moment for backend to start
sleep 3

# Start frontend server
echo "Starting frontend server on port 5173..."
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

echo -e "\n✅ Development servers started!"
echo "   Backend:  http://localhost:8080"
echo "   Frontend: http://localhost:5173"
echo -e "\nPress Ctrl+C to stop all servers\n"

# Wait for user to stop
wait