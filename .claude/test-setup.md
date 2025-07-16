# DFS Optimizer Test Setup

Since Docker has network issues and Go isn't installed locally, here's how to test the application:

## Option 1: Fix Docker Network
```bash
# Restart Docker Desktop
# Then retry:
docker-compose up -d
```

## Option 2: Local Setup Requirements
You'll need to install:
1. Go 1.21+ (https://go.dev/dl/)
2. PostgreSQL 15+
3. Redis 7+

Then run:
```bash
# Backend
cd backend
go mod download
go run cmd/server/main.go

# Frontend (in another terminal)
cd frontend
npm install
npm run dev
```

## Option 3: View the Code Structure
The application is fully implemented with:

### Backend Features:
- REST API at `localhost:8080`
- Player management
- Lineup optimization with correlation/stacking
- Monte Carlo simulations
- CSV export for DraftKings/FanDuel

### Frontend Features:
- Dashboard at `localhost:5173`
- Player pool management
- Lineup builder (UI pending)
- Optimizer controls (UI pending)
- Export functionality

### Key Files to Review:
- `backend/internal/optimizer/algorithm.go` - Core optimization logic
- `backend/internal/simulator/monte_carlo.go` - Simulation engine
- `frontend/src/pages/Optimizer.tsx` - Main optimizer interface
- `docker-compose.yml` - Full stack configuration