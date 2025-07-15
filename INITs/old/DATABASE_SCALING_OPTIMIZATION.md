## FEATURE:

Database & Application Scaling Optimization - Performance improvements for DFS lineup optimizer to handle large-scale operations efficiently.

## EXAMPLES:

Current inefficiencies identified:
- **N+1 Query Problem**: Loading 106 lineups with 636 lineup_players requires 107+ database queries
- **Missing Indexes**: Query like `SELECT * FROM lineups WHERE user_id = 1 AND contest_id = 3 AND is_submitted = false` lacks compound index
- **Optimization Algorithm**: Backtracking generates up to 10,000 combinations with deep copying and no memoization
- **Data Aggregation**: Sequential processing of parallel fetch results with inefficient player merging

## DOCUMENTATION:

- **Database Schema**: Tables analyzed include `lineups`, `lineup_players`, `players`, `contests`
- **Current Indexes**: Limited indexes on individual columns, missing compound indexes for common query patterns
- **Optimization Algorithm**: `backend/internal/optimizer/algorithm.go` uses recursive backtracking with exponential complexity
- **Data Aggregation**: `backend/internal/services/aggregator.go` processes multiple data providers sequentially after parallel fetch
- **Lineup Handling**: `backend/internal/api/handlers/lineup.go` loads players individually for each lineup

## OTHER CONSIDERATIONS:

### Critical Performance Issues:
1. **Database Design**: 
   - N+1 query problem in `LoadPlayers()` method
   - Missing compound indexes for `(user_id, contest_id, is_submitted)` pattern
   - `PlayerPositions` stored in memory instead of normalized in database

2. **Algorithm Scalability**:
   - Exponential complexity with no early pruning
   - Deep copying of entire lineup objects during generation
   - No memoization for repeated calculations

3. **Data Flow Bottlenecks**:
   - Sequential processing despite parallel data fetching
   - No connection pooling optimization
   - Cache invalidation clears entire cache on any change

### Implementation Phases:
- **Phase 1**: Database optimization (80-90% query reduction expected)
- **Phase 2**: Algorithm optimization (60-70% performance improvement)
- **Phase 3**: Data flow improvements (50-60% API response time improvement)
- **Phase 4**: Monitoring and observability

### Common AI Assistant Gotchas:
- Don't just add indexes everywhere - analyze query patterns first
- Avoid premature optimization - measure before and after changes
- Consider memory vs. CPU tradeoffs in algorithm optimizations
- Cache invalidation is harder than cache population
- Database migrations need to be backwards compatible
- Load testing required to validate performance improvements

### Specific Requirements:
- Maintain backwards compatibility with existing API
- Ensure data integrity during schema changes
- Add proper error handling for new bulk operations
- Consider transaction boundaries for bulk updates
- Monitor query performance impact of new indexes