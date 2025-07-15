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

**Microservices Architecture (Go) - Production-Grade DFS Engine**

**API Gateway (`services/api-gateway/`)**
- Central entry point using Gin framework with JWT authentication
- Request routing and load balancing to specialized services
- CORS handling and request/response logging
- WebSocket proxy for real-time optimization progress
- Health checks and circuit breaker patterns

**User Service (`services/user-service/`)**
- Phone-based authentication with OTP verification using Supabase
- User management, preferences, and subscription handling
- JWT token generation and validation
- SMS integration (Twilio/Supabase) for verification codes
- Supabase PostgreSQL database for user data

**Golf Service (`services/golf-service/`)**
- **Multi-Sport Provider System**: RapidAPI (golf), ESPN, BallDontLie, TheSportsDB with intelligent fallbacks
- **Rate Limiting & Caching**: Redis-backed caching with aggressive RapidAPI rate limiting (20 req/day)
- Tournament data synchronization and player statistics
- Weather integration and course condition tracking
- **Supabase PostgreSQL**: All golf data stored in unified Supabase database

**Optimization Service (`services/optimization-service/`)**
- **Optimization Engine**: Advanced knapsack algorithm with correlation/stacking support
- **Monte Carlo Simulator**: Parallel worker pools for contest outcome simulation
- **Real-time WebSocket Hub**: Live optimization progress and player updates
- **Supabase PostgreSQL**: All optimization and lineup data stored in unified Supabase database

**Simplified Database Strategy (Supabase-First)**
- **Supabase PostgreSQL**: ALL data - users, golf, contests, players, optimization results, lineups, Stripe subscriptions
- **Redis**: Cross-service caching, rate limiting, session management only
- **Local PostgreSQL**: REMOVED - migrated to Supabase for unified data management

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

**Microservices (Go)**
```bash
# Start entire microservices stack
docker-compose up -d

# Start individual services for development
cd services/api-gateway && go run cmd/server/main.go
cd services/user-service && go run cmd/server/main.go
cd services/golf-service && go run cmd/server/main.go
cd services/optimization-service && go run cmd/server/main.go

# Install dependencies for all services
cd services/api-gateway && go mod download
cd services/user-service && go mod download
cd services/golf-service && go mod download
cd services/optimization-service && go mod download

# Run tests for specific services
cd services/golf-service && go test ./...
cd services/optimization-service && go test ./...
cd services/user-service && go test ./...

# Build services for production
cd services/api-gateway && go build -o server cmd/server/main.go
cd services/user-service && go build -o server cmd/server/main.go
cd services/golf-service && go build -o server cmd/server/main.go
cd services/optimization-service && go build -o server cmd/server/main.go

# Database migrations (service-specific)
cd services/user-service && go run ../../backend/cmd/migrate/main.go up
cd services/golf-service && go run ../../backend/cmd/migrate/main.go up
cd services/optimization-service && go run ../../backend/cmd/migrate/main.go up

# Lint all services
cd services/api-gateway && golangci-lint run
cd services/user-service && golangci-lint run
cd services/golf-service && golangci-lint run
cd services/optimization-service && golangci-lint run
```

**Legacy Backend (Monolith - for reference)**
```bash
# Navigate to legacy backend
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

**Docker (Simplified Microservices)**
```bash
# Run entire microservices stack (PostgreSQL removed - uses Supabase)
docker-compose up -d

# Only Redis and microservices - no local database containers
# Services automatically connect to Supabase PostgreSQL

# View logs for all services
docker-compose logs -f

# View logs for specific services
docker-compose logs -f api-gateway
docker-compose logs -f user-service
docker-compose logs -f golf-service
docker-compose logs -f optimization-service

# Stop services
docker-compose down

# Rebuild specific service
docker-compose build user-service
docker-compose up -d user-service

# Check service health (all connect to Supabase)
docker-compose ps
curl http://localhost:8080/health   # API Gateway
curl http://localhost:8081/health   # Golf Service
curl http://localhost:8082/health   # Optimization Service
curl http://localhost:8083/health   # User Service
```

**Local Development (Alternative to Docker)**
```bash
# Start all services locally (requires Go 1.21+, Redis only - PostgreSQL via Supabase)
./start-local.sh

# Stop all local services
./stop-local.sh

# Note: No local PostgreSQL needed - all services connect to Supabase
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

**Microservices API Gateway (`services/api-gateway/`)**
- **Base path**: `/api/v1/` for all client-facing endpoints
- **Entry point**: API Gateway at port 8080 (nginx proxy on port 80/443)
- **Authentication**: JWT with service-specific middleware and token validation
- **WebSocket**: `/ws/optimization-progress/:user_id` (proxied to optimization service)

**Service Boundaries & Routing**

**Authentication Routes (User Service Proxy)**
```
POST /api/v1/auth/register     → user-service:8083/auth/register
POST /api/v1/auth/login        → user-service:8083/auth/login  
POST /api/v1/auth/verify       → user-service:8083/auth/verify
POST /api/v1/auth/resend       → user-service:8083/auth/resend
GET  /api/v1/auth/me          → user-service:8083/auth/me
POST /api/v1/auth/refresh     → user-service:8083/auth/refresh
POST /api/v1/auth/logout      → user-service:8083/auth/logout
```

**User Management Routes (User Service Proxy)**
```
ANY /api/v1/users/*           → user-service:8083/users/*
```

**Golf Data Routes (Golf Service Proxy)**
```
ANY /api/v1/sports/*          → golf-service:8081/sports/*
ANY /api/v1/contests/*        → golf-service:8081/contests/*
ANY /api/v1/golf/*            → golf-service:8081/golf/*
```

**Optimization Routes (Optimization Service Proxy)**
```
ANY /api/v1/optimize          → optimization-service:8082/optimize
ANY /api/v1/optimize/*        → optimization-service:8082/optimize/*
ANY /api/v1/simulate/*        → optimization-service:8082/simulate/*
```

**Lineup Management (API Gateway Handled)**
```
GET    /api/v1/lineups        → api-gateway (local handler)
POST   /api/v1/lineups        → api-gateway (local handler)
GET    /api/v1/lineups/:id    → api-gateway (local handler)
PUT    /api/v1/lineups/:id    → api-gateway (local handler)
DELETE /api/v1/lineups/:id    → api-gateway (local handler)
POST   /api/v1/lineups/:id/export → api-gateway (local handler)
```

**Health & Monitoring**
```
GET /health                   → api-gateway health check
GET /ready                    → api-gateway readiness check
GET /status/services          → all service health status
GET /status/circuit-breakers  → circuit breaker status
```

**Database Models & Service Ownership (Unified Supabase)**
- **All Services (Supabase PostgreSQL)**: All tables in single unified database
  - **User Data**: User, UserPreferences, SubscriptionTier, StripeCustomers, StripeSubscriptions
  - **Sports Data**: Sports, Players, Contests (across all sports)
  - **Golf Data**: GolfTournaments, GolfPlayerEntries, GolfRoundScores, GolfCourseHistory
  - **DFS Data**: Lineups, LineupPlayers, OptimizationResults, SimulationResults
  - **Payment Data**: StripeEvents, payment processing tables

### 🧱 Code Structure & Conventions

**Microservices Structure**
```
services/
├── api-gateway/           # Central API Gateway
│   ├── cmd/server/main.go    # Gateway entry point
│   ├── internal/
│   │   ├── api/handlers/     # Gateway-specific handlers
│   │   ├── middleware/       # Auth, CORS, logging middleware
│   │   ├── proxy/           # Service proxy logic
│   │   └── websocket/       # WebSocket hub
│   └── config/nginx.conf    # NGINX configuration
├── user-service/          # User Authentication & Management
│   ├── cmd/server/main.go    # User service entry point
│   ├── internal/
│   │   ├── api/handlers/     # Auth, user endpoints
│   │   ├── models/          # User, preferences models
│   │   └── services/        # User business logic, SMS
│   └── migrations/          # User-specific migrations
├── golf-service/          # Golf Data & Providers
│   ├── cmd/server/main.go    # Golf service entry point
│   ├── internal/
│   │   ├── api/handlers/     # Golf endpoints
│   │   ├── models/          # Golf-specific models
│   │   ├── providers/       # RapidAPI, ESPN, etc.
│   │   └── services/        # Data fetching, caching
│   └── migrations/          # Golf-specific migrations
├── optimization-service/  # Optimization & Simulation
│   ├── cmd/server/main.go    # Optimization service entry point
│   ├── internal/
│   │   ├── api/handlers/     # Optimization endpoints
│   │   ├── optimizer/       # Optimization algorithms
│   │   ├── simulator/       # Monte Carlo simulation
│   │   └── websocket/       # Real-time progress
│   └── migrations/          # Optimization-specific migrations
└── shared/                # Shared packages across services
    ├── pkg/
    │   ├── config/          # Configuration management
    │   ├── database/        # Database connections
    │   ├── logger/          # Structured logging
    │   └── optimizer/       # Shared optimization logic
    └── types/               # Common type definitions
```

**Legacy Backend Structure (Monolith)**
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

**Service-Specific Configuration**

**API Gateway (Port 8080)**
```bash
SERVICE_NAME=gateway
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/dfs_optimizer
REDIS_URL=redis://localhost:6379/2
JWT_SECRET=your-secret-key-change-in-production
CORS_ORIGINS=http://localhost:5173,http://localhost:3000,http://localhost:80
GOLF_SERVICE_URL=http://golf-service:8081
OPTIMIZATION_SERVICE_URL=http://optimization-service:8082
USER_SERVICE_URL=http://user-service:8083
ENV=development
```

**User Service (Port 8083) - Supabase Integration**
```bash
SERVICE_NAME=user
PORT=8083
DATABASE_URL=postgresql://postgres:[password]@db.[project-id].supabase.co:5432/postgres
REDIS_URL=redis://localhost:6379/3
JWT_SECRET=your-secret-key-change-in-production
SUPABASE_URL=https://[project-id].supabase.co
SUPABASE_SERVICE_KEY=your-service-role-key
SUPABASE_ANON_KEY=your-anon-key
SMS_PROVIDER=supabase  # or twilio
TWILIO_ACCOUNT_SID=your-twilio-sid
TWILIO_AUTH_TOKEN=your-twilio-token
TWILIO_FROM_NUMBER=+1234567890
ENV=development
```

**Golf Service (Port 8081)**
```bash
SERVICE_NAME=golf
PORT=8081
DATABASE_URL=postgres://postgres:postgres@localhost:5432/dfs_optimizer
REDIS_URL=redis://localhost:6379/0
RAPIDAPI_KEY=your-rapidapi-key
ESPN_RATE_LIMIT=10
CIRCUIT_BREAKER_THRESHOLD=5
ENABLE_BACKGROUND_JOBS=true
DATA_FETCH_INTERVAL=30m
ENV=development
```

**Optimization Service (Port 8082)**
```bash
SERVICE_NAME=optimization
PORT=8082
DATABASE_URL=postgres://postgres:postgres@localhost:5432/dfs_optimizer
REDIS_URL=redis://localhost:6379/1
MAX_LINEUPS=150
OPTIMIZATION_TIMEOUT=30
MAX_SIMULATIONS=100000
SIMULATION_WORKERS=4
ENV=development
```

**Frontend Configuration**
```bash
# React + Vite Frontend
VITE_API_URL=http://localhost:8080/api/v1  # API Gateway endpoint
VITE_WS_URL=ws://localhost:8080/ws          # WebSocket endpoint
VITE_SUPABASE_URL=https://[project-id].supabase.co
VITE_SUPABASE_ANON_KEY=your-anon-key
```

**External Data Provider APIs**
- `RAPIDAPI_KEY`: RapidAPI Live Golf Data API key ⚠️ **Critical**: Basic plan = 20 requests/day limit
- `BALLDONTLIE_API_KEY`: NBA player and game data (free tier available)
- `THESPORTSDB_API_KEY`: Multi-sport data provider (free tier: "1")
- `ESPN_RATE_LIMIT`: ESPN scraping rate limit (requests per hour)
- `DATA_FETCH_INTERVAL`: How often to sync external data (e.g., "30m", "1h")

**Supabase Configuration**
- `SUPABASE_URL`: Your Supabase project URL
- `SUPABASE_SERVICE_KEY`: Service role key for server-side operations
- `SUPABASE_ANON_KEY`: Anonymous key for client-side operations
- `SMS_PROVIDER`: Choose between "supabase" or "twilio" for OTP delivery

**Performance & Rate Limiting**
- `MAX_LINEUPS`: Maximum lineups per optimization (default: 150)
- `OPTIMIZATION_TIMEOUT`: Optimization timeout in seconds (default: 30)
- `MAX_SIMULATIONS`: Maximum Monte Carlo simulations (default: 100000)
- `SIMULATION_WORKERS`: Parallel workers (default: 4, adjust based on CPU cores)
- `CIRCUIT_BREAKER_THRESHOLD`: Failure threshold for external APIs (default: 5)

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

**Hybrid Database Architecture**

**Supabase PostgreSQL (User Service)**
- **Database**: Managed Supabase instance for user data
- **Purpose**: User accounts, authentication, preferences, subscriptions
- **Connection**: Via Supabase client libraries and direct PostgreSQL connection
- **Tables**: `auth.users`, `public.users`, `user_preferences`, `subscription_tiers`
- **Features**: Built-in auth, real-time subscriptions, row-level security (RLS)
- **Setup**: See `supabase-setup-guide.md` for configuration instructions

**Local PostgreSQL (DFS Data)**
- **Database**: `dfs_optimizer` 
- **Purpose**: Golf data, contests, players, optimization results, lineups
- **Default credentials**: postgres/postgres
- **Tables**: `golf_tournaments`, `golf_players`, `contests`, `lineups`, `optimization_results`
- **Migrations**: Service-specific migrations in each service's `migrations/` directory

**Unified Database Configuration (Simplified)**
```bash
# ALL SERVICES → Supabase PostgreSQL (Single Database)
DATABASE_URL=postgresql://postgres:[password]@db.jkltmqniqbwschxjogor.supabase.co:5432/postgres

# User Service: Users, auth, preferences, Stripe data
# Golf Service: Golf tournaments, players, courses  
# Optimization Service: Optimization results, simulations
# API Gateway: Lineups, session management
```

**Database Setup (Supabase-First)**
```bash
# Run consolidated schema in Supabase SQL Editor
# Execute: supabase-consolidated-schema.sql

# Contains all tables for:
# - User management + Stripe integration
# - Golf tournaments and player data  
# - DFS contests and lineup optimization
# - Monte Carlo simulation results
```

**Redis Configuration**
- **Purpose**: Cross-service caching, rate limiting, session management
- **Service DB Allocation**:
  - DB 0: Golf Service (data caching)
  - DB 1: Optimization Service (result caching)
  - DB 2: API Gateway (session management)
  - DB 3: User Service (auth tokens)

### 🚧 Implementation Status & Critical Issues

**Microservices Backend**: ✅ Production-Ready (90% Complete)
- ✅ **API Gateway**: Fully operational with service routing and health checks
- ✅ **User Service**: Complete phone auth with Supabase integration and OTP verification
- ✅ **Golf Service**: Multi-provider data integration with rate limiting and caching
- ✅ **Optimization Service**: Advanced optimization and simulation engines complete
- ✅ **Database Architecture**: Hybrid Supabase + PostgreSQL strategy implemented
- ✅ **Authentication Flow**: End-to-end phone auth through API Gateway working
- ✅ **Real-time Updates**: WebSocket hub for optimization progress operational
- ⚠️ **MINOR**: Service discovery could be enhanced with load balancing
- ⚠️ **MINOR**: Background job scheduling needs monitoring and alerting

**Legacy Backend**: ⚠️ Maintained for Reference (85% Complete)
- ✅ All monolithic features preserved in `backend/` directory
- ✅ Can be used as fallback during migration testing
- ⚠️ Not recommended for new development (use microservices)

**Frontend**: ⚠️ Infrastructure Complete, Features Pending (40% Complete)
- ✅ Complete TypeScript setup with React Query and Zustand
- ✅ Catalyst UI Kit integration and TailwindCSS configuration
- ✅ **Authentication Flow**: Complete phone auth integration with API Gateway
- ✅ **User Preferences**: Dashboard loads user preferences from Supabase
- ✅ **Session Management**: Automatic token refresh and session persistence
- ❌ **MISSING**: Drag-and-drop lineup builder implementation
- ❌ **MISSING**: Real-time WebSocket integration for live optimization updates
- ❌ **MISSING**: Simulation visualization components
- ❌ **MISSING**: Manual lineup construction and editing

**Authentication Architecture**: ✅ Complete (100%)
- ✅ **Phone Registration**: OTP-based registration with conflict detection
- ✅ **Login Flow**: Separate login endpoint for existing users
- ✅ **Token Management**: JWT generation and validation across services
- ✅ **Session Persistence**: Frontend auth store with automatic refresh
- ✅ **User Preferences**: Sync between Supabase and frontend state

**Authentication Flow (End-to-End)**
```
1. Frontend → API Gateway → User Service (phone registration/login)
2. User Service → Supabase (OTP generation and SMS delivery)
3. User → Frontend (OTP verification)
4. Frontend → API Gateway → User Service (OTP verification)
5. User Service → JWT token generation → Frontend
6. Frontend → Automatic token refresh (every 50 minutes)
7. All subsequent API calls include JWT header through API Gateway
```

**Critical Path to MVP**: Golf data integration → Drag-and-drop UI → Real-time updates

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

**Microservices Issues**

**Service Communication Problems**
- Check all services are running: `docker-compose ps`
- Verify service URLs in API Gateway configuration:
  ```bash
  curl http://localhost:8081/health  # Golf Service
  curl http://localhost:8082/health  # Optimization Service  
  curl http://localhost:8083/health  # User Service
  ```
- Check API Gateway routing: `curl http://localhost:8080/status/services`
- Verify JWT secret consistency across all services

**Authentication Issues**
- **Supabase Connection**: Check `SUPABASE_URL` and `SUPABASE_SERVICE_KEY` in user-service
- **JWT Token Problems**: Ensure same `JWT_SECRET` in API Gateway and User Service
- **Phone Auth Not Working**: 
  - Verify SMS provider configuration in user-service
  - Check Supabase phone auth is enabled in dashboard
  - Validate phone number format (E.164: +1234567890)
- **Token Refresh Failing**: Check API Gateway auth middleware configuration

**Database Connection Issues (Supabase-First)**
- **Unified Supabase Database (All Services)**: 
  - Verify connection string format: `postgresql://postgres:[password]@db.jkltmqniqbwschxjogor.supabase.co:5432/postgres`
  - Check project ID and password in `DATABASE_URL`
  - Test connection: `psql "postgresql://postgres:xk3StS7e@S!Crcj@db.jkltmqniqbwschxjogor.supabase.co:5432/postgres"`
  - Ensure schema is deployed: Run `supabase-consolidated-schema.sql` in Supabase SQL Editor
- **Local PostgreSQL**: REMOVED - no longer used

**Redis Connection Issues**
- Check Redis is running: `docker-compose ps redis`
- Verify each service uses correct Redis DB:
  - Golf Service: DB 0
  - Optimization Service: DB 1  
  - API Gateway: DB 2
  - User Service: DB 3
- Test Redis connection: `redis-cli ping`

**Frontend Issues**
- **API Connection**: Check `VITE_API_URL` points to API Gateway (port 8080)
- **WebSocket Issues**: Verify `VITE_WS_URL` points to API Gateway WebSocket endpoint
- **Authentication**: Check `VITE_SUPABASE_URL` and `VITE_SUPABASE_ANON_KEY`
- **CORS Errors**: Verify `CORS_ORIGINS` in API Gateway includes frontend URL

**Performance Issues**
- **Optimization Timeouts**: Increase `OPTIMIZATION_TIMEOUT` in optimization-service
- **Slow Golf Data**: Check `RAPIDAPI_KEY` quota and rate limits
- **High Memory Usage**: Reduce `MAX_LINEUPS` and `SIMULATION_WORKERS`
- **Service Startup Slow**: Check `ENABLE_BACKGROUND_JOBS=false` for faster startup

**Docker Issues (Simplified Stack)**
- **Services Not Starting**: Check Docker Compose configuration (PostgreSQL container removed)
- **Health Check Failing**: Verify health check endpoints return 200 status and can connect to Supabase
- **Network Issues**: Ensure all services are on the same Docker network (`dfs-network`)
- **Supabase Connection**: Verify all services can reach Supabase PostgreSQL externally

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

**Microservices Core**
- `services/api-gateway/` - Central API Gateway with routing and auth middleware
- `services/user-service/` - Phone authentication and user management with Supabase
- `services/golf-service/` - Golf data providers and tournament synchronization
- `services/optimization-service/` - Optimization algorithms and Monte Carlo simulation
- `shared/` - Shared packages and types across all services
- `services/*/migrations/` - Service-specific database migrations

**API Gateway**
- `services/api-gateway/internal/proxy/` - Service proxy and load balancing
- `services/api-gateway/internal/middleware/` - Auth, CORS, and logging middleware
- `services/api-gateway/internal/websocket/` - WebSocket hub for real-time updates

**User Service**
- `services/user-service/internal/models/` - User, preferences, and subscription models
- `services/user-service/internal/services/` - SMS service and user business logic
- `services/user-service/migrations/` - Supabase-compatible user schema migrations

**Golf Service**
- `services/golf-service/internal/providers/` - RapidAPI, ESPN, external data providers
- `services/golf-service/internal/services/` - Data fetching, caching, weather integration

**Optimization Service**
- `services/optimization-service/internal/optimizer/` - Core optimization algorithms
- `services/optimization-service/internal/simulator/` - Monte Carlo simulation engine
- `services/optimization-service/internal/websocket/` - Real-time progress updates

**Legacy Backend (Reference)**
- `backend/internal/api/handlers/` - All monolithic API endpoint handlers
- `backend/internal/optimizer/` - Original optimization algorithms
- `backend/internal/simulator/` - Original Monte Carlo simulation engine
- `backend/migrations/` - Complete database schema for monolithic deployment
- `backend/tests/` - Integration tests with manual test checklists

**Frontend Core**
- `src/pages/` - Main application pages (Dashboard, Optimizer, Lineups)
- `src/components/auth/` - Phone authentication components
- `src/services/` - API clients for microservices architecture
- `src/store/` - Zustand stores for auth and preferences with Supabase integration
- `src/types/` - TypeScript type definitions for all entities

**Configuration & Deployment**
- `docker-compose.yml` - Complete microservices stack with health checks
- `docker-compose.microservices.yml` - Microservices-specific deployment
- `docker-compose.monolith.yml` - Legacy monolith deployment option
- `supabase-setup-guide.md` - Step-by-step Supabase configuration guide
- Each service uses Viper for configuration management with environment variables
- Frontend uses Vite with API Gateway proxy configuration

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
