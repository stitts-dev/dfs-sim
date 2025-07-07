# DFS Lineup Optimizer

A full-stack Daily Fantasy Sports (DFS) lineup optimizer with Go backend and React frontend, featuring Monte Carlo simulations, optimization algorithms with correlation/stacking, and multi-sport support.

## 🚀 Features

- **Multi-Sport Support**: NBA, NFL, MLB, NHL
- **Advanced Optimization**: Knapsack algorithm with position constraints and correlation-based stacking
- **Monte Carlo Simulations**: Simulate contest outcomes with player correlations
- **Real-time Updates**: WebSocket support for live optimization progress
- **Platform Support**: Export lineups for DraftKings and FanDuel
- **Modern Tech Stack**: Go + React with TypeScript

## 🛠️ Tech Stack

**Backend:**
- Go with Gin framework
- PostgreSQL with GORM ORM
- Redis for caching
- WebSocket for real-time updates

**Frontend:**
- React 18 with TypeScript
- Vite for fast development
- TailwindCSS for styling
- React Query for data fetching

## 🏃‍♂️ Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose (optional)

### Using Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/dfs-optimizer.git
cd dfs-optimizer

# Copy environment variables
cp .env.example .env

# Start all services
docker-compose up

# Run database migrations (in another terminal)
docker-compose exec backend go run cmd/migrate/main.go up

# Seed sample data
docker-compose exec backend go run cmd/migrate/main.go seed
```

The application will be available at:
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

### Manual Setup

#### Backend Setup

```bash
cd backend

# Install dependencies
go mod download

# Set up environment variables
cp ../.env.example .env

# Run database migrations
go run cmd/migrate/main.go up

# Seed sample data (optional)
go run cmd/migrate/main.go seed

# Start the server
go run cmd/server/main.go
```

#### Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

## 📚 Project Structure

```
.
├── backend/
│   ├── cmd/                # Application entrypoints
│   ├── internal/           # Private application code
│   │   ├── api/           # HTTP handlers and routes
│   │   ├── models/        # Database models
│   │   ├── optimizer/     # Optimization algorithms
│   │   └── simulator/     # Monte Carlo simulation
│   └── pkg/               # Public packages
├── frontend/
│   ├── src/
│   │   ├── components/    # React components
│   │   ├── pages/         # Page components
│   │   ├── services/      # API services
│   │   └── types/         # TypeScript types
│   └── public/            # Static assets
└── examples/              # Example implementations
```

## 🎮 Usage

1. **Select a Contest**: Choose from available DFS contests on the dashboard
2. **Build Your Lineup**: Use the drag-and-drop interface or optimizer
3. **Configure Optimization**: Set correlation weights, stacking rules, and player locks
4. **Run Simulations**: Test your lineups with Monte Carlo simulations
5. **Export Lineups**: Download CSV files for DraftKings/FanDuel upload

## 🧪 Development

### Running Tests

```bash
# Backend tests
cd backend
go test ./...

# Frontend tests
cd frontend
npm test
```

### Linting

```bash
# Backend
golangci-lint run

# Frontend
npm run lint
```

## 📖 API Documentation

The API is available at `http://localhost:8080/api/v1`. Key endpoints:

- `GET /contests` - List available contests
- `GET /contests/:id/players` - Get players for a contest
- `POST /optimize` - Generate optimized lineups
- `POST /simulate` - Run Monte Carlo simulations
- `POST /export` - Export lineups to CSV

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.