## FEATURE:

Build a full-stack Daily Fantasy Sports (DFS) lineup optimizer that replicates SaberSim's core functionality. The system should include:

**Core Features:**
1. **Multi-Sport Support**: Start with NBA, then expand to NFL, MLB, NHL
2. **Lineup Optimization Engine**: 
   - Knapsack algorithm for salary cap optimization
   - Correlation/stacking capabilities with adjustable slider
   - Position constraints and lineup rules per platform
   - Multi-lineup generation (up to 150 lineups)
   - Diversity controls to avoid duplicate lineups

3. **Monte Carlo Simulation Engine**:
   - Simulate games thousands of times for variance modeling
   - Player performance distributions based on projections
   - Correlation-based outcome generation
   - Contest simulation (GPP vs Cash optimization)
   - Real-time simulation progress visualization

4. **Data Management**:
   - Player pool with stats, projections, salaries, ownership
   - Support for DraftKings and FanDuel formats
   - Real-time updates when lineups/injuries change
   - CSV import/export for lineups

5. **User Interface**:
   - Dashboard with sport/contest selection
   - Drag-and-drop lineup builder
   - Player search/filter with advanced criteria
   - Correlation slider and stacking controls
   - Simulation visualizer showing progress
   - Results view with lineup stats and export

**Technical Requirements:**
- Backend: Go with Gin framework, PostgreSQL, Redis, WebSockets
- Frontend: React with TypeScript, TailwindCSS, Recharts
- Real-time updates via WebSocket
- JWT authentication
- Docker deployment

## EXAMPLES:

Place the following examples in the `examples/` folder:

1. **optimizer_algorithm.go** - Basic knapsack implementation with position constraints
2. **correlation_matrix.go** - Player correlation calculations for stacking
3. **monte_carlo_sim.go** - Simple Monte Carlo simulation for player outcomes
4. **lineup_builder_component.tsx** - React component for drag-drop lineup building
5. **api_client.ts** - TypeScript API client with proper typing
6. **docker-compose.yml** - Multi-service setup for local development

These examples should demonstrate:
- Go error handling patterns
- Proper TypeScript typing
- WebSocket implementation
- Database query optimization
- React component structure

## DOCUMENTATION:

1. **DFS Platform Rules**:
   - DraftKings contest rules and constraints
   - FanDuel lineup requirements
   - Salary cap and position requirements by sport

2. **Algorithm References**:
   - Knapsack optimization algorithms
   - Monte Carlo simulation techniques
   - Correlation matrix calculations
   - Linear programming for lineup optimization

3. **API Documentation**:
   - Sports data APIs for player stats
   - WebSocket best practices
   - Redis caching strategies

4. **UI/UX References**:
   - React DnD documentation
   - TailwindCSS components
   - Recharts for data visualization
   - Responsive design patterns

## OTHER CONSIDERATIONS:

1. **Performance Requirements**:
   - Optimization should complete within 5 seconds for 150 lineups
   - Support concurrent optimization requests
   - Cache optimization results for 15 minutes
   - Handle 10,000+ players in player pool

2. **Data Accuracy**:
   - Validate salary cap constraints
   - Ensure position requirements are met
   - Prevent invalid lineup submissions
   - Handle platform-specific rules (FLEX, utility spots)

3. **Common AI Pitfalls to Avoid**:
   - Don't assume player data format - each sport is different
   - Correlation calculations vary by sport (batting order vs game script)
   - Platform rules differ between DraftKings and FanDuel
   - Optimization must consider both ceiling and floor projections
   - WebSocket connections need proper error handling and reconnection

4. **Security Considerations**:
   - Never expose optimization algorithms in frontend
   - Rate limit API endpoints to prevent abuse
   - Validate all lineup submissions server-side
   - Implement proper CORS for API

5. **Testing Requirements**:
   - Unit tests for optimization algorithms
   - Integration tests for full lineup generation
   - Load tests for concurrent users
   - Validate lineup exports match platform formats

6. **MVP Focus**:
   - Start with NBA only
   - Basic correlation without advanced stacking
   - 20 lineup maximum for MVP
   - Simple ownership projections
   - Focus on DraftKings format first