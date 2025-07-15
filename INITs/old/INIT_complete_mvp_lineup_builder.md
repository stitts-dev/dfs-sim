# INIT: Complete MVP Lineup Builder

## Task Overview
Complete the MVP lineup builder functionality to achieve a fully functional DFS optimization platform. This task focuses on implementing the core user interaction features that are currently missing from the otherwise production-ready backend infrastructure.

## Context
The DFS optimization platform is 75% complete with excellent microservices architecture, complete authentication system, and robust optimization algorithms. However, critical user interaction features are missing that prevent this from being a usable MVP.

## Current State Analysis
- ✅ **Backend Infrastructure**: 90% complete with all 4 microservices operational
- ✅ **Authentication System**: Complete phone-based auth with Supabase integration
- ✅ **Database Architecture**: Unified Supabase PostgreSQL with Redis caching
- ✅ **Optimization Algorithms**: Advanced knapsack with correlation matrices and stacking
- ❌ **Frontend User Experience**: 60% complete with key interactions missing

## Critical MVP Gaps
1. **Drag-and-Drop Lineup Builder**: Infrastructure exists but key interactions missing
2. **Real-Time WebSocket Integration**: Backend ready, frontend client missing
3. **Lineup Export/Import**: No CSV generation for DraftKings/FanDuel
4. **Advanced Player Filtering**: Basic search but no advanced criteria

## Implementation Tasks

### Phase 1: Core Lineup Building (Priority: CRITICAL)

#### Task 1: Complete Drag-and-Drop Lineup Builder
**Files**: `frontend/src/components/LineupBuilder/`, `frontend/src/components/PlayerPool/`
**Timeline**: 1-2 weeks

**User Stories**:
1. As a user, I want to drag players from the pool to lineup positions so I can build lineups intuitively
2. As a user, I want real-time salary cap validation so I know when I'm over budget
3. As a user, I want position validation so I can't place players in wrong positions
4. As a user, I want to filter/search players by name, team, price, projections
5. As a user, I want to see player stats and projections when building lineups

**Technical Requirements**:
- Implement `@dnd-kit` for position management
- Real-time position validation with error messages
- Live salary cap tracking with warnings at 95% capacity
- Advanced player filtering with multiple criteria
- Player stats popup cards with projections and recent performance

**Acceptance Criteria**:
- [ ] Players can be dragged from pool to lineup positions
- [ ] Invalid position drops are rejected with clear feedback
- [ ] Salary cap updates in real-time with color-coded warnings
- [ ] Player search works across name, team, position, price range
- [ ] Player stats display on hover/click with loading states

#### Task 2: Real-Time WebSocket Integration
**Files**: `frontend/src/services/websocket.ts`, `frontend/src/hooks/useWebSocket.ts`
**Timeline**: 1 week

**User Stories**:
1. As a user, I want to see live optimization progress so I know the system is working
2. As a user, I want real-time player updates during contests so my data is current
3. As a user, I want automatic reconnection if my connection drops
4. As a user, I want to see when other users are optimizing (social proof)

**Technical Requirements**:
- WebSocket client with automatic reconnection logic
- Real-time progress bars for optimization and simulation
- Live player news, injury updates, weather changes
- Graceful handling of disconnections and reconnections

**Acceptance Criteria**:
- [ ] WebSocket connects on app load and maintains connection
- [ ] Optimization progress updates in real-time with percentage complete
- [ ] Player updates appear instantly without page refresh
- [ ] Connection automatically recovers from network interruptions
- [ ] Error states are handled gracefully with user feedback

#### Task 3: Lineup Export/Import System
**Files**: `frontend/src/services/export.ts`, `frontend/src/components/LineupExport/`
**Timeline**: 1 week

**User Stories**:
1. As a user, I want to export lineups to CSV so I can upload to DraftKings/FanDuel
2. As a user, I want to compare multiple lineups side-by-side
3. As a user, I want to save favorite lineups for later use
4. As a user, I want to see lineup diversity metrics across my portfolio

**Technical Requirements**:
- CSV export generation for DraftKings/FanDuel compatible files
- Side-by-side lineup comparison with overlap percentages
- Lineup storage with metadata (contest, date, performance)
- Portfolio analytics with diversity metrics, correlation analysis, exposure tracking

**Acceptance Criteria**:
- [ ] Export generates valid CSV files for major DFS platforms
- [ ] Lineup comparison shows player overlap and correlation metrics
- [ ] Saved lineups persist across sessions with proper metadata
- [ ] Portfolio view shows exposure percentages and diversity scores

### Phase 2: Advanced Features (Priority: HIGH)

#### Task 4: Advanced Optimization UI
**Files**: `frontend/src/components/OptimizationConfig/`, `frontend/src/components/CorrelationMatrix/`
**Timeline**: 2 weeks

**User Stories**:
1. As a user, I want to configure team stacking rules so I can optimize for specific strategies
2. As a user, I want to see correlation matrices visually so I understand player relationships
3. As a user, I want to set player exposure limits so I don't over-rely on specific players
4. As a user, I want to create multiple lineup variations with different strategies

**Technical Requirements**:
- Visual interface for team/game stack rules
- Heat maps showing player relationships
- Sliders for min/max player exposure across lineups
- Pre-configured optimization strategies (GPP, Cash, Balanced)

#### Task 5: Enhanced Player Analysis
**Files**: `frontend/src/components/PlayerAnalysis/`, `frontend/src/services/analytics.ts`
**Timeline**: 1-2 weeks

**User Stories**:
1. As a user, I want to see player trends over time so I can make informed decisions
2. As a user, I want weather impact analysis for golf contests
3. As a user, I want to see player ownership projections in GPP contests
4. As a user, I want course history and performance metrics

**Technical Requirements**:
- Charts showing player performance over time
- Course conditions and player historical performance
- Estimated ownership percentages for GPP strategy
- Historical performance at specific courses

### Phase 3: Testing & Performance (Priority: MEDIUM)

#### Task 6: Comprehensive Testing Suite
**Files**: `frontend/src/**/__tests__/`, `services/*/tests/`
**Timeline**: 1 week

**Testing Requirements**:
- Unit tests for all optimization algorithms and business logic
- Integration tests for end-to-end user flows
- Performance tests for optimization service load testing
- WebSocket tests for real-time feature testing

#### Task 7: Performance Optimization
**Files**: Frontend bundle optimization, database indexing
**Timeline**: 1 week

**Performance Requirements**:
- Frontend bundle code splitting and lazy loading
- Database query optimization with proper indexing
- Reduce optimization service memory footprint
- Improve cache hit rates and invalidation

## Success Metrics

### MVP Success Criteria
- [ ] **User Onboarding**: 90% of users complete lineup building tutorial
- [ ] **Optimization Usage**: 80% of users generate optimized lineups
- [ ] **Export Usage**: 60% of users export lineups to DFS platforms
- [ ] **Real-Time Engagement**: 50% of users actively use during optimization
- [ ] **Performance**: <5 second optimization time for 150 lineups

### Phase 2 Success Criteria
- [ ] **Advanced Features**: 40% of users use stacking configuration
- [ ] **Analytics Usage**: 30% of users analyze player trends
- [ ] **Portfolio Management**: 50% of users save multiple lineups
- [ ] **User Retention**: 70% weekly retention rate

## Technical Architecture

### Key Components
- **LineupBuilder**: Main drag-and-drop interface
- **PlayerPool**: Filterable player selection interface
- **WebSocket Client**: Real-time communication layer
- **Export Service**: CSV generation and lineup management
- **OptimizationConfig**: Advanced configuration interface
- **PlayerAnalysis**: Analytics and trend visualization

### Data Flow
1. User authenticates via phone (✅ Complete)
2. User selects contest from available tournaments (✅ Complete)
3. User builds lineup via drag-and-drop (❌ Missing)
4. User configures optimization parameters (❌ Missing)
5. System optimizes lineups with real-time progress (❌ Missing)
6. User exports lineups to DFS platforms (❌ Missing)

## Implementation Priority

### Week 1: Core Lineup Building
1. **Implement drag-and-drop lineup builder** - Complete the @dnd-kit integration
2. **Add real-time salary cap validation** - Live updates with color-coded warnings
3. **Enhance player filtering** - Advanced search with multiple criteria
4. **Implement WebSocket client** - Frontend client with reconnection logic

### Week 2: Export & Polish
1. **Implement lineup export** - CSV generation for DraftKings/FanDuel
2. **Add real-time progress tracking** - Live optimization progress bars
3. **Complete lineup comparison** - Side-by-side analysis tools
4. **Polish user experience** - Error handling and loading states

## Quality Gates
- [ ] All new features have unit tests
- [ ] Performance benchmarks maintained
- [ ] Security review for new endpoints
- [ ] User experience validation

## Expected Outcomes
Upon completion of this task, the DFS optimization platform will be a fully functional MVP with:
- Intuitive drag-and-drop lineup building
- Real-time optimization progress
- Professional lineup export capabilities
- Advanced player analysis tools
- Comprehensive testing coverage

The platform will be ready for user testing and production deployment, providing a compelling alternative to existing DFS optimization tools like SaberSim.

## Next Steps
After MVP completion, focus areas include:
1. Advanced stacking configuration
2. Player ownership projections
3. Multi-sport expansion
4. Mobile optimization
5. Payment integration activation