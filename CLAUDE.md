# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project: SaberSim Clone - DFS Lineup Optimizer

A full-stack Daily Fantasy Sports (DFS) lineup optimizer with Go backend and React frontend, replicating SaberSim's core functionality including Monte Carlo simulations, lineup optimization with correlation/stacking, and multi-sport support.

### 📋 Context Engineering Resources

This project uses Context Engineering principles. Key resources:
- **PRP Blueprint**: `PRPs/dfs-lineup-optimizer.md` - Comprehensive implementation guide with code structure, algorithms, and validation loops
- **Custom Commands**: `.claude/commands/` - Generate and execute PRPs
- **Initial Requirements**: `INITIAL.md` - Original feature specifications
- **Testing Guide**: `test-setup.md` - Setup options and implementation status

### 🏗️ Architecture Overview

**Backend (Go)**
- REST API using Gin framework (`backend/`)
- Simulation engine for Monte Carlo game simulations
- Optimizer engine with correlation/stacking algorithms
- PostgreSQL database with GORM ORM
- WebSocket support for real-time updates
- Redis for caching optimization results

**Frontend (React + TypeScript)**
- Dashboard with sport/contest selection (`frontend/src/pages/`)
- Player pool management with filtering/search
- Drag-and-drop lineup builder
- Optimizer controls (correlation slider, stacking options)
- Real-time simulation visualizer
- Export functionality for DFS platforms

### 🚀 Development Commands

**Backend (Go)**
```bash
# Navigate to backend
cd backend

# Install dependencies
go mod download

# Run development server
go run cmd/server/main.go

# Run tests
go test ./...

# Build for production
go build -o bin/server cmd/server/main.go

# Database migrations
go run cmd/migrate/main.go up
go run cmd/migrate/main.go down

# Lint (ensure golangci-lint is installed)
golangci-lint run
```

**Frontend (React)**
```bash
# Navigate to frontend
cd frontend

# Install dependencies
npm install

# Run development server
npm run dev

# Run tests
npm test

# Build for production
npm run build

# Lint and type check
npm run lint
npm run type-check
```

**Docker**
```bash
# Run entire stack
docker-compose up

# Run in background
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

**Local Development (Alternative to Docker)**
```bash
# Start all services locally (requires Go 1.21+, PostgreSQL, Redis)
./start-local.sh

# Stop all local services
./stop-local.sh
```

**Testing Single Components**
```bash
# Run a single Go test
cd backend
go test -run TestOptimizer ./internal/optimizer/...

# Run a single React component test
cd frontend
npm test -- --testNamePattern="LineupBuilder"

# Debug with verbose output
go test -v ./internal/api/...
```

### 🧱 Code Structure & Conventions

**Backend Structure**
```
backend/
├── cmd/
│   ├── server/main.go      # API server entry point
│   └── migrate/main.go     # Database migration tool
├── internal/
│   ├── api/               # HTTP handlers and routes
│   ├── models/            # Database models
│   ├── optimizer/         # Optimization algorithms
│   ├── simulator/         # Monte Carlo simulation engine
│   └── services/          # Business logic
├── pkg/                   # Shared packages
└── tests/                 # Integration tests
```

**Frontend Structure**
```
frontend/
├── src/
│   ├── components/        # Reusable UI components
│   ├── pages/            # Page components
│   ├── hooks/            # Custom React hooks
│   ├── services/         # API client services
│   ├── store/            # State management (Redux/Zustand)
│   └── types/            # TypeScript type definitions
└── tests/                # Component and integration tests
```

### 📎 Style & Conventions

**Go Backend**
- Follow standard Go conventions and effective Go guidelines
- Use structured logging with `zerolog` or `logrus`
- Handle errors explicitly, never ignore them
- Use context for request-scoped values and cancellation
- Keep handlers thin, business logic in services
- Use dependency injection for testability

**React Frontend**
- Use functional components with TypeScript
- Follow React hooks best practices
- Use TailwindCSS for styling
- Implement proper error boundaries
- Use React Query or SWR for data fetching
- Keep components focused and composable

### 🧪 Testing Requirements

**Backend Tests**
- Unit tests for all business logic
- Integration tests for API endpoints
- Mock external dependencies
- Test coverage target: 80%
- Use table-driven tests where appropriate

**Frontend Tests**
- Component tests with React Testing Library
- Integration tests for critical user flows
- Mock API responses
- Accessibility tests
- Visual regression tests for key components

### 🔐 Security & Performance

- Never commit sensitive data or API keys
- Use environment variables for configuration
- Implement rate limiting on API endpoints
- Cache optimization results in Redis
- Use database indexes for frequent queries
- Implement proper CORS configuration
- Validate all user inputs
- Use prepared statements for database queries

### ⚙️ Environment Configuration

Key environment variables (see `.env.example`):
- `DATABASE_URL`: PostgreSQL connection (default: postgres://postgres:postgres@localhost:5432/dfs_optimizer)
- `REDIS_URL`: Redis connection (default: redis://localhost:6379)
- `JWT_SECRET`: Authentication secret key
- `CORS_ORIGINS`: Multiple origins supported (comma-separated)
- `MAX_LINEUPS`: Maximum lineups per optimization (default: 150)
- `OPTIMIZATION_TIMEOUT`: Timeout in seconds (default: 30)
- `MAX_SIMULATIONS`: Max Monte Carlo simulations (default: 100000)
- `SIMULATION_WORKERS`: Parallel simulation workers (default: 4)
- `BALLDONTLIE_API_KEY`: NBA data API key
- `THESPORTSDB_API_KEY`: Sports data API key
- `RAPIDAPI_KEY`: RapidAPI Live Golf Data API key (Basic plan: 20 req/day)

### 🔄 Project Awareness & Context

- Check project documentation before implementing features
- Follow existing patterns in the codebase
- Update documentation when adding new features
- Keep code modular and maintainable
- Use meaningful commit messages

### 📚 Key Algorithms & Features

**Optimization Engine**
- Knapsack algorithm for salary cap optimization
- Correlation matrix for player relationships
- Stacking rules (game stacks, team stacks, mini stacks)
- Position constraints and lineup rules
- Multi-lineup generation with diversity

**Simulation Engine**
- Monte Carlo simulations for game outcomes
- Player performance distributions
- Correlation-based outcome generation
- Contest simulation (GPP vs Cash)
- Ownership projection integration

**Data Management**
- Player stats and projections
- Historical performance data
- Real-time lineup updates
- Contest rules and constraints
- Export formats (CSV for DraftKings/FanDuel)

**Golf Data Provider (RapidAPI)**
- Live tournament data and leaderboards
- Player statistics and performance metrics
- Aggressive caching for Basic plan (20 req/day limit)
- Automatic fallback to ESPN Golf when API limit reached
- Rate limit tracking with daily/monthly counters
- Cache warming strategy for optimal API usage

### 📊 Database Setup

- Database name: `dfs_optimizer`
- Default credentials: postgres/postgres
- Run migrations with seed data: `go run cmd/migrate/main.go up`
- WebSocket support for real-time optimization progress
- JWT authentication for API endpoints

### 🚧 Implementation Status

**Backend**: ✅ Fully implemented
- All API endpoints operational
- Optimization and simulation engines complete
- Database models and migrations ready
- WebSocket real-time updates working
- External API integrations functional

**Frontend**: ⚠️ Partial implementation
- ✅ Infrastructure, routing, and state management
- ✅ Authentication and user preferences
- ⚠️ Lineup builder drag-and-drop UI (pending)
- ⚠️ Optimizer controls UI (pending)
- ⚠️ Some component implementations incomplete
- See `test-setup.md` for detailed status

### 🔔 System Management 

- Restart services on checkpoint landmarks
- Don't restart frontend on service restart or testing checkpoint

### 🎯 Common Tasks & Workflows

**Adding a New Sport**
1. Add sport type to `backend/internal/models/sport.go`
2. Create provider in `backend/internal/providers/`
3. Add contest rules in `backend/internal/optimizer/rules/`
4. Update frontend sport selector in `frontend/src/components/SportSelector.tsx`

**Modifying Optimization Algorithm**
1. Core logic: `backend/internal/optimizer/optimizer.go`
2. Correlation matrix: `backend/internal/optimizer/correlation.go`
3. Stacking rules: `backend/internal/optimizer/stacking.go`
4. Test with: `go test ./internal/optimizer/...`

**Working with External APIs**
- Providers in `backend/internal/providers/`
- Add new provider by implementing `DataProvider` interface
- Use existing providers (BallDontLie, TheSportsDB) as templates

### 🛠️ Troubleshooting

**Database Connection Issues**
- Ensure PostgreSQL is running: `docker-compose ps`
- Check connection string in `.env`
- Run migrations: `go run cmd/migrate/main.go up`

**Frontend Not Loading**
- Check API URL in `frontend/.env`
- Verify CORS origins in backend `.env`
- Check browser console for errors

**Optimization Timeouts**
- Increase `OPTIMIZATION_TIMEOUT` in `.env`
- Reduce `MAX_LINEUPS` for faster results
- Check Redis connection for caching issues

### 📝 Development Notes

- **Container/Logs Workflow**
  - I'll test container/logs manually just tell me when to re-build/restart em