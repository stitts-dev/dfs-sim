# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project: SaberSim Clone - DFS Lineup Optimizer

A full-stack Daily Fantasy Sports (DFS) lineup optimizer with Go backend and React frontend, replicating SaberSim's core functionality including Monte Carlo simulations, lineup optimization with correlation/stacking, and multi-sport support.

## 🏗️ High-Level Architecture

### Microservices Architecture
The project follows a **microservices architecture** with service separation by domain:

```
API Gateway (8080) ← Frontend (5173)
├── Sports Data Service (8081)     # Player data, contests, tournaments
├── Optimization Service (8082)     # Lineup optimization, algorithms
├── User Service (8083)            # Authentication, user management
├── AI Recommendations (8084)      # AI-powered insights
└── Realtime Service (8085)        # Live data, notifications
```

### Technology Stack

**Backend (Go 1.21)**
- **Framework**: Gin (HTTP routing)
- **Database**: PostgreSQL via Supabase + GORM ORM
- **Cache**: Redis (separate DB per service)
- **Authentication**: Supabase Auth + JWT
- **WebSockets**: Gorilla WebSocket
- **Advanced Libraries**: 
  - Gorgonia (ML/Neural Networks)
  - Gonum (Mathematical computations)
  - Circuit breaker, rate limiting patterns

**Frontend (React 18 + TypeScript)**
- **Build Tool**: Vite
- **UI Framework**: Tailwind CSS + Headless UI
- **State Management**: Zustand
- **Data Fetching**: React Query
- **Authentication**: Supabase client
- **Charts**: Recharts
- **Routing**: React Router DOM

**Infrastructure**
- **Containerization**: Docker + Docker Compose
- **Load Balancer**: NGINX
- **Database**: Supabase (PostgreSQL)
- **Caching**: Redis (multi-database setup)

## 📁 Key Directory Structure

```
/
├── services/                    # Microservices (Go)
│   ├── api-gateway/            # Central API gateway, request routing
│   ├── sports-data-service/    # External data providers (ESPN, DataGolf)
│   ├── optimization-service/   # Core DFS algorithms, Monte Carlo
│   ├── user-service/          # Auth, user management, subscriptions
│   ├── ai-recommendations-service/ # Claude API integration
│   └── realtime-service/      # WebSockets, live updates
├── shared/                     # Shared Go packages
│   ├── pkg/                   # Common utilities (config, DB, logger)
│   └── types/                 # Shared type definitions
├── frontend/                   # React SPA
│   ├── src/components/        # UI components
│   ├── src/services/          # API clients
│   ├── src/store/            # State management
│   └── src/types/            # TypeScript definitions
├── scripts/                    # Development/deployment scripts
└── docker-compose.yml         # Multi-service orchestration
```

## 🔧 Common Development Commands

### Docker Environment (Recommended)
```bash
# Start all services
docker-compose up --build

# Start specific services
docker-compose up api-gateway sports-data-service

# View logs
docker-compose logs -f optimization-service

# Stop all services
docker-compose down
```

### Development Scripts
```bash
# Start local development environment
./scripts/start-dev.sh

# Run backend tests
./scripts/test-backend.sh

# Test authentication flow
./scripts/test-auth-flow.sh

# Validate full integration
./scripts/validate-full-integration.sh

# Deploy database schema
./scripts/deploy-schema.sh
```

### Service-Specific Commands
```bash
# Run individual service
cd services/optimization-service
go run cmd/server/main.go

# Run tests
go test ./...

# Frontend development
cd frontend
npm run dev
npm run build
npm run type-check
```

## 🗄️ Database Schema & Models

### Core DFS Tables
- **players** (22 columns): Cross-sport player data with projections, salaries, ownership, injury status
- **contests** (23 columns): DFS contest information (DraftKings, FanDuel) with roster positions, prize pools
- **lineups** (21 columns): User-generated and optimized lineups with simulation results
- **lineup_players** (8 columns): Junction table for lineup-player relationships
- **simulation_results** (14 columns): Monte Carlo simulation outputs with portfolio analysis

### User Management
- **users** (13 columns): Extends Supabase auth with subscription tiers, usage tracking
- **user_preferences** (13 columns): UI settings, sport/platform preferences, tutorial state
- **subscription_tiers** (11 columns): Free/Basic/Premium tier definitions
- **stripe_customers** (7 columns): Stripe integration for payments
- **stripe_subscriptions** (12 columns): Subscription management
- **phone_verification_codes** (7 columns): SMS verification support

### Golf Analytics Tables
- **golf_tournaments** (20 columns): Tournament data with course info, weather, field strength
- **course_analytics** (18 columns): Course difficulty, skill premiums, historical scoring
- **strokes_gained_history** (19 columns): DataGolf integration with SG metrics
- **player_course_fits** (15 columns): Player-course compatibility analysis
- **weather_impact_tracking** (14 columns): Weather effect on performance

### Advanced Analytics
- **algorithm_performance** (21 columns): Algorithm testing and optimization results
- **correlation_matrices** (12 columns): Player correlation data for stacking
- **golf_strategy_effectiveness** (21 columns): Strategy performance tracking
- **optimization_results** (10 columns): Optimization algorithm outputs

### Database Development with MCP

#### MCP Supabase Reader Tools
Use the following MCP tools for database operations:

```bash
# Check database health and connection
mcp__supabase-reader__health_check

# List all tables in the database
mcp__supabase-reader__list_tables

# Get detailed table schema
mcp__supabase-reader__describe_table --table_name players

# Execute read-only queries (auto-limited to 1000 rows)
mcp__supabase-reader__execute_query --query "SELECT * FROM users LIMIT 5"

# Check connection pool status
mcp__supabase-reader__pool_status
```

#### Database Connection Details
- **Database**: PostgreSQL via Supabase
- **Connection**: Read-only via MCP server
- **Security**: All queries are enforced read-only
- **Limits**: 1000 rows max per query
- **Connection Pool**: 10 max connections

### Migration Pattern
Each service manages its own migrations in `services/[service]/migrations/`

### Current Schema Status
- **Total Tables**: 22 base tables
- **Advanced Analytics**: Integrated DataGolf API support
- **Real-time Features**: Event tracking and notifications
- **Golf Specialization**: Course analytics and strokes gained data

## 🔌 API Structure & Endpoints

### API Gateway (Port 8080)
Acts as the central entry point, routing requests to appropriate services:

```
/api/v1/auth/*          → user-service
/api/v1/users/*         → user-service
/api/v1/golf/*          → sports-data-service
/api/v1/optimization/*  → optimization-service
/api/v1/ai/*           → ai-recommendations-service
/api/v1/realtime/*     → realtime-service
```

### Key Service Endpoints

**Sports Data Service (8081)**
- `GET /contests` - Available DFS contests
- `GET /players/{contest_id}` - Player pool for contest
- `GET /golf/tournaments` - Golf tournament data
- `GET /projections/{sport}` - Player projections

**Optimization Service (8082)**
- `POST /optimize` - Generate optimized lineups
- `POST /simulate` - Monte Carlo simulation
- `GET /analytics/{user_id}` - User optimization analytics
- `POST /golf/optimize` - Golf-specific optimization

**User Service (8083)**
- `POST /auth/register` - User registration
- `POST /auth/login` - Phone/email authentication
- `GET /auth/me` - Current user info
- `PUT /preferences` - Update user preferences

## 🎨 Frontend Architecture

### Component Structure
```
src/components/
├── auth/              # Authentication components
├── LineupBuilder/     # Drag-and-drop lineup builder
├── OptimizerControls/ # Optimization settings UI
├── SimulationViz/     # Monte Carlo visualization
├── realtime/          # Live data components
└── ui/               # Reusable UI components
```

### State Management Pattern
- **Zustand stores** for different concerns:
  - `unifiedAuth.ts` - Authentication state
  - `preferences.ts` - User preferences
  - `realtime.ts` - Live data updates

### Service Layer
- **apiClient.ts** - Unified API client with auto-retry
- **auth.ts** - Authentication service
- **supabase.ts** - Supabase client configuration
- **websocketService.ts** - Real-time communication

## 🔄 Development Workflow Patterns

### Service Communication
- **Synchronous**: HTTP requests via service proxy
- **Asynchronous**: Redis pub/sub for events
- **Real-time**: WebSocket connections through realtime-service

### Database Development Workflow
1. **Schema Analysis**: Use MCP tools to explore current database structure
2. **Migration Creation**: Create migration files in `services/[service]/migrations/`
3. **Schema Deployment**: Use `./scripts/deploy-schema.sh` for manual deployment
4. **Verification**: Use MCP tools to verify schema changes

#### Database Schema Management
```bash
# Analyze current schema
mcp__supabase-reader__list_tables
mcp__supabase-reader__describe_table --table_name [table_name]

# Create new migration files
# services/[service]/migrations/[number]_[description].sql

# Deploy schema changes
./scripts/deploy-schema.sh

# Verify deployment
mcp__supabase-reader__health_check
```

### Error Handling
- **Circuit breaker pattern** for external API calls
- **Structured logging** with logrus
- **Graceful degradation** for non-critical features

### Testing Strategy
- **Unit tests**: Individual algorithm components
- **Integration tests**: Service-to-service communication
- **End-to-end tests**: Full user workflows
- **Database tests**: Use MCP tools for data validation

### Authentication Flow
1. **Phone/Email registration** via Supabase
2. **OTP verification** through SMS/email
3. **JWT token management** with auto-refresh
4. **Session persistence** across services

## 🎯 Core Business Logic

### DFS Optimization Algorithm
Located in `services/optimization-service/internal/optimizer/`:
- **Dynamic Programming** optimization
- **Correlation matrices** for player relationships
- **Stacking rules** (team, game, mini-stacks)
- **Exposure management** across multiple lineups

### Monte Carlo Simulation
- **Variance modeling** for player projections
- **Contest simulation** with payout structures
- **Portfolio analysis** across multiple lineups
- **Risk/reward metrics** (Sharpe ratio, cash rate)

### Golf-Specific Features
- **Cut probability modeling**
- **Tee time correlation**
- **Course-specific adjustments**
- **Strokes gained analytics**

## 🛠 Development Techniques

- `use @.env` - Utility for loading environment variables dynamically across services
- `use @docker-compose.yml` - Utility for managing multi-container Docker application configurations
- `dont dev in @backend.deprecated/` - Only reference deprecated backend code, do not develop within this directory
- `backend @services/` - Backend code and services located in the services directory
- `use supabase mcp to make sure this project has correct schema`
- 3rd part providers also should have type safety
- When fixing errors, if decided to remove core logic for simpler placeholder, add detailed todo in its place along with temp fix (avoid when possible)

## 🔍 Key Integration Points

### External Data Providers
- **ESPN API**: General sports data
- **DataGolf API**: Advanced golf analytics
- **RapidAPI**: Supplementary data sources
- **OpenWeather**: Weather data for golf

### Third-Party Services
- **Supabase**: Database and authentication
- **Claude API**: AI recommendations
- **Twilio**: SMS for OTP (alternative to Supabase)
- **Stripe**: Payment processing (disabled in current build)

## 📊 Performance Considerations

### Optimization Service
- **Resource limits**: 2 CPUs, 2GB RAM
- **Optimization timeout**: 30 seconds
- **Max simulations**: 100,000 iterations
- **Worker pools**: 4 simulation workers

### Caching Strategy
- **Redis per service**: Separate databases (0-5)
- **Cache expiration**: 3600s for AI recommendations
- **Background jobs**: 30-minute data refresh intervals

### Database Optimization
- **Connection pooling** via GORM
- **Indexed queries** on frequently accessed fields
- **JSONB columns** for flexible metadata storage

## 🚀 Getting Started

1. **Clone the repository**
2. **Set up environment variables** (see .env.example)
3. **Start services**: `docker-compose up --build`
4. **Frontend development**: `cd frontend && npm run dev`
5. **Access the application**: http://localhost:5173

The application automatically proxies API requests to the gateway at localhost:8080, which routes them to the appropriate microservices.

## 📝 Notes for Development

- **Always use absolute paths** when referencing files
- **Prefer editing existing files** over creating new ones
- **Follow the microservices pattern** - don't mix concerns across services
- **Use shared types** from the `shared/types` package
- **Implement proper error handling** with circuit breakers
- **Add structured logging** for debugging
- **Test integrations thoroughly** across service boundaries