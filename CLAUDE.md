# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project: SaberSim Clone - DFS Lineup Optimizer

A full-stack Daily Fantasy Sports (DFS) lineup optimizer with Go backend and React frontend, replicating SaberSim's core functionality including Monte Carlo simulations, lineup optimization with correlation/stacking, and multi-sport support.

### 📋 Context Engineering Resources

This project uses Context Engineering principles with comprehensive documentation:
- **PRP Blueprints**: `PRPs/` directory - Implementation guides with algorithms and validation
- **Initial Tasks**: `INITs/` directory - Feature specifications and optimization tasks
- **Context Engineering Template**: `templates/` - Base templates and patterns
- **Testing Guide**: `test-setup.md` - Setup options and implementation status
- **API Test Documentation**: Various `test-*-api.md` files for testing specific endpoints

### 🏗️ Architecture Overview

**Backend (Go) - Production-Grade DFS Engine**
- REST API using Gin framework with JWT authentication (`backend/`)
- **Optimization Engine**: Advanced knapsack algorithm with correlation/stacking support
- **Monte Carlo Simulator**: Parallel worker pools for contest outcome simulation
- **Multi-Sport Provider System**: RapidAPI (golf), ESPN, BallDontLie, TheSportsDB with intelligent fallbacks
- **Rate Limiting & Caching**: Redis-backed caching with aggressive RapidAPI rate limiting (20 req/day)
- **Real-time WebSocket Hub**: Live optimization progress and player updates
- PostgreSQL with GORM, comprehensive migrations and constraints

**Frontend (React + TypeScript) - Modern DFS Interface**
- **Catalyst UI Kit Integration**: Tailwind Plus components (vendorized in `src/catalyst/`)
- **State Management**: React Query for server state, Zustand for client state
- **Drag-and-Drop**: `@dnd-kit` and `react-beautiful-dnd` for lineup building
- **Real-time Updates**: WebSocket integration for live optimization progress
- **Multi-Platform Export**: CSV generation for DraftKings/FanDuel upload
- ⚠️ **Implementation Status**: Infrastructure complete, UI components partially implemented

### 🧮 Core Algorithms & Data Flow

**Optimization Engine (`internal/optimizer/`)**
- **Primary Algorithm**: Modified knapsack with position constraint validation
- **Correlation Matrix**: Player relationship scoring for game/team stacks
- **Stacking Rules**: Team stacks, game stacks, mini stacks, QB stacks with configurable weights
- **Lineup Diversity**: Multi-lineup generation with minimum player difference requirements
- **Position Flexibility**: FLEX/UTIL position handling with eligibility mapping

**Monte Carlo Simulation (`internal/simulator/`)**
- **Worker Pool Architecture**: Parallel simulation execution with configurable worker count
- **Correlated Outcomes**: Player performance generation using correlation matrices
- **Contest Modeling**: GPP vs Cash game simulation with different scoring distributions
- **Statistical Analysis**: ROI calculation, percentile analysis, ownership projections

**External Data Pipeline (`internal/providers/`)**
- **Provider Interface**: Unified abstraction for all sports data sources
- **Rate Limiting Strategy**: Per-provider limits with Redis-backed counters
- **Fallback Chain**: Primary → Secondary → Cache for data availability
- **Data Aggregation Service**: Combines multiple sources with conflict resolution

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

# Run integration tests
go test ./tests/...

# Run specific API tests
go test -v ./tests/api_optimizer_test.go
go test -v ./tests/golf_integration_test.go

# Run with coverage
go test -cover ./...
```

### 🌐 API Architecture & Domain Boundaries

**REST API Structure (`internal/api/`)**
- Base path: `/api/v1/` for all endpoints
- Authentication: JWT with optional/required middleware per route group
- WebSocket: `/ws` (separate from REST API, no versioning)

**Core Domain Handlers**
- **Contest Management**: `/contests/*` - Discovery, sync, data fetching
- **Player Operations**: `/players/*` - Player details, statistics, history
- **Lineup Management**: `/lineups/*` - CRUD operations, submission
- **Optimization**: `/optimize`, `/optimize/validate` - Core optimization endpoints
- **Simulation**: `/simulate/*` - Monte Carlo simulation execution
- **Golf Integration**: `/golf/*` - Tournament data, leaderboards, projections
- **AI Recommendations**: `/ai/*` - Claude integration for lineup suggestions
- **Export**: `/export/*` - CSV generation for DFS platforms

**Database Models & Relationships**
- **Core Entities**: Player, Contest, Lineup, SimulationResult
- **Sport-Specific**: GolfTournament, GolfPlayer with performance history
- **User Data**: UserPreferences with beginner mode and optimization settings
- **Metadata**: Position requirements, platform constraints, team mappings

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
│   ├── catalyst/          # Catalyst UI Kit (Tailwind Plus) components (vendorized)
│   ├── templates/         # Tailwind Plus Commit template code (reference/demo)
│   ├── pages/             # Page components
│   ├── hooks/             # Custom React hooks
│   ├── services/          # API client services
│   ├── store/             # State management (Redux/Zustand)
│   └── types/             # TypeScript type definitions
└── tests/                # Component and integration tests
```

### 🧩 Tailwind Plus & Catalyst UI Kit Integration

**Catalyst UI Kit**
- All Catalyst UI Kit components are located in `src/catalyst/` (or `libs/catalyst-ui-kit/` if using as a library).
- Import and use as needed in your React components:
  ```tsx
  import { Button } from '@/catalyst/Button'
  // or, if in libs:
  import { Button } from '@/libs/catalyst-ui-kit/Button'
  ```
- Prefer extending/wrapping for custom behavior; avoid direct edits to vendor code unless documented.
- Track the original version/source in a `README.md` inside the vendor directory.

**Tailwind Plus Commit Template**
- Template/demo code is stored in `src/templates/commit/` (or `/templates/tailwind-plus-commit/` if outside src).
- Use as a reference for advanced layouts, animation, or best-practice patterns.
- Copy code as needed into your own components, and adapt to your app’s needs.
- Keep template/demo code separate from production code.

**Best Practices**
- Never edit vendor code directly unless you document the change.
- Wrap or extend vendor components for custom behavior.
- Keep vendor code up to date by tracking the original source and noting the version in a local README.
- Document any usage patterns or gotchas for your team.

**Updating Vendor Code**
- If a new version of Catalyst or Commit is released, replace the contents of the relevant directory.
- Document any local changes in a `README.md` inside the vendor directory.

**References**
- [Tailwind Plus UI Blocks Documentation](https://tailwindcss.com/plus/ui-blocks/documentation)
- [Catalyst UI Kit](https://tailwindcss.com/plus/ui-kit)
- [Tailwind Plus License](https://tailwindcss.com/plus/license)

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

### ⚙️ Environment Configuration & External Dependencies

**Core System Configuration**
- `DATABASE_URL`: PostgreSQL connection (default: postgres://postgres:postgres@localhost:5432/dfs_optimizer)
- `REDIS_URL`: Redis connection for caching and rate limiting (default: redis://localhost:6379)
- `JWT_SECRET`: Authentication secret key for API access
- `CORS_ORIGINS`: Multiple frontend origins supported (comma-separated)
- `PORT`: Backend server port (default: 8080)
- `ENV`: Environment mode (development/production)

**Optimization & Performance Limits**
- `MAX_LINEUPS`: Maximum lineups per optimization request (default: 150)
- `OPTIMIZATION_TIMEOUT`: Timeout in seconds for optimization requests (default: 30)
- `MAX_SIMULATIONS`: Maximum Monte Carlo simulations per request (default: 100000)
- `SIMULATION_WORKERS`: Parallel simulation workers (default: 4, adjust based on CPU cores)

**External Data Provider APIs**
- `RAPIDAPI_KEY`: RapidAPI Live Golf Data API key ⚠️ **Critical**: Basic plan = 20 requests/day limit
- `BALLDONTLIE_API_KEY`: NBA player and game data (free tier available)
- `THESPORTSDB_API_KEY`: Multi-sport data provider (free tier: "1")
- `ESPN_RATE_LIMIT`: ESPN scraping rate limit (requests per hour)
- `DATA_FETCH_INTERVAL`: How often to sync external data (e.g., "1h", "30m")

**AI Integration**
- `ANTHROPIC_API_KEY`: Claude AI integration for lineup recommendations
- `AI_RATE_LIMIT`: Claude API requests per hour limit
- `AI_CACHE_EXPIRATION`: Cache expiration for AI responses (seconds)

### 🔄 Project Awareness & Context

- Check project documentation before implementing features
- Follow existing patterns in the codebase
- Update documentation when adding new features
- Keep code modular and maintainable
- Use meaningful commit messages

### 📚 Key Algorithms & Features

**Optimization Engine (`internal/optimizer/`)**
- **Core Algorithm**: Dynamic knapsack solver with position constraints in `algorithm.go`
- **Correlation Matrix**: Player relationship calculations in `correlation.go` and `golf_correlation.go`
- **Stacking Engine**: Game stacks, team stacks, mini stacks in `stacking.go`
- **Constraint System**: Position requirements and salary cap validation in `constraints.go`
- **Golf-Specific**: Tee time correlations, cut line probability, weather impact

**Monte Carlo Simulator (`internal/simulator/`)**
- **Parallel Processing**: Worker pool architecture with configurable workers
- **Correlated Outcomes**: Player performance relationships with shared variance
- **Contest Modeling**: GPP vs Cash game payout structures in `contest.go`
- **Distribution Engine**: Normal, log-normal, beta distributions in `distributions.go`
- **Performance Analytics**: Percentiles, ROI, variance analysis

**Provider Architecture (`internal/providers/`)**
- **Interface-Driven**: Common `DataProvider` interface for all sports APIs
- **RapidAPI Golf**: Rate-limited client with daily quota tracking (`rapidapi_golf.go`)
- **ESPN Fallback**: Free tier backup for golf data (`espn_golf.go`)
- **Multi-Sport Support**: NBA (BallDontLie), general sports (TheSportsDB)
- **Intelligent Caching**: Redis-backed with TTL and cache warming strategies

**Real-time Architecture**
- **WebSocket Hub**: Concurrent connection management in `services/websocket.go`
- **Progress Reporting**: Live optimization and simulation progress updates
- **Event Broadcasting**: Player updates, contest changes, system notifications

### 📊 Database Setup

- Database name: `dfs_optimizer`
- Default credentials: postgres/postgres
- Run migrations with seed data: `go run cmd/migrate/main.go up`
- WebSocket support for real-time optimization progress
- JWT authentication for API endpoints

### 🚧 Implementation Status & Critical Issues

**Backend**: ✅ Production-Ready (85% Complete)
- ✅ All API endpoints operational with comprehensive handlers
- ✅ Advanced optimization and simulation engines complete
- ✅ Database models with proper constraints and migrations
- ✅ WebSocket real-time updates working
- ✅ Multi-provider external API integrations
- ⚠️ **CRITICAL**: API routing misconfiguration - `/api/v1/*` routes not accessible
- ⚠️ **BLOCKING**: Startup operations can take 30+ seconds due to synchronous API calls
- ⚠️ Performance optimization needed for large player pools (>150 players)

**Frontend**: ⚠️ Infrastructure Complete, Features Pending (40% Complete)
- ✅ Complete TypeScript setup with React Query and Zustand
- ✅ Catalyst UI Kit integration and TailwindCSS configuration
- ✅ Authentication flow and user preferences system
- ❌ **MISSING**: Drag-and-drop lineup builder implementation
- ❌ **MISSING**: Real-time WebSocket integration for live updates
- ❌ **MISSING**: Simulation visualization components
- ❌ **MISSING**: Manual lineup construction and editing

**Critical Path to MVP**: Fix API routing → Basic data integration → Drag-and-drop UI

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

### 🔧 Development Scripts & Utilities

**Quick Setup Scripts**
```bash
# Start all services locally
./start-local.sh

# Start development environment
./start-dev.sh

# Test backend only
./test-backend.sh
```

**Backend Utility Commands**
```bash
# Check data integrity
cd backend && go run cmd/check-data/main.go

# Migration commands
cd backend && go run cmd/migrate/main.go up
cd backend && go run cmd/migrate/main.go down
cd backend && go run cmd/migrate/main.go seed
```

**Testing Scripts**
```bash
# Test AI recommendations
backend/scripts/test-ai-recommendations.sh

# Test golf integration
backend/scripts/test-golf-integration.sh
```

### 📂 Key Directories & Files

**Backend Core**
- `internal/api/handlers/` - All API endpoint handlers
- `internal/optimizer/` - Core optimization algorithms with correlation and stacking
- `internal/simulator/` - Monte Carlo simulation engine
- `internal/providers/` - External data providers (BallDontLie, ESPN, RapidAPI)
- `internal/services/` - Business logic layer
- `migrations/` - Database schema evolution
- `tests/` - Integration tests with manual test checklists

**Frontend Core**
- `src/pages/` - Main application pages (Dashboard, Optimizer, Lineups)
- `src/components/` - Reusable UI components with AI integration
- `src/services/` - API clients and authentication
- `src/types/` - TypeScript type definitions for all entities

**Configuration & Setup**
- `docker-compose.yml` - Complete stack deployment with health checks
- `start-local.sh` - Local development setup with dependency validation
- Backend uses Viper for configuration management with environment variable support
- Frontend uses Vite with proxy configuration for API calls

### 🔧 Critical Architecture Patterns

**Provider Interface Pattern (`internal/providers/`)**
All external data providers implement a common interface. When adding new sports:
```go
type DataProvider interface {
    GetContests(ctx context.Context) ([]Contest, error)
    GetPlayers(ctx context.Context, contestID string) ([]Player, error)
    GetPlayerStats(ctx context.Context, playerID string) (*PlayerStats, error)
}
```

**Optimization Pipeline (`internal/optimizer/algorithm.go`)**
Core optimization follows this pattern:
1. **Constraint Validation**: Check salary cap, position requirements
2. **Correlation Matrix Build**: Calculate player relationships
3. **Lineup Generation**: Knapsack-based optimization with diversity
4. **Stacking Application**: Apply team/game stack rules
5. **Exposure Management**: Ensure player exposure limits

**Rate Limiting Strategy (Critical for RapidAPI)**
- **Request Queue System**: Priority-based queuing (high/medium/low)
- **Cache Warming**: Pre-fetch tournament data at 00:01 UTC
- **Fallback Hierarchy**: RapidAPI → ESPN → TheSportsDB → Cache
- **Circuit Breakers**: Graceful degradation when APIs unavailable

**WebSocket Architecture (`services/websocket.go`)**
Real-time updates use a hub pattern:
- **Connection Manager**: Concurrent-safe client management
- **Event Broadcasting**: Type-safe message distribution
- **Progress Reporting**: Live optimization/simulation updates
