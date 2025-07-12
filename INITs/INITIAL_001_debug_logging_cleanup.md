## FEATURE:

Remove debug logging statements and implement proper structured logging throughout the optimization algorithm

## EXAMPLES:

Current problematic code in `backend/internal/optimizer/algorithm.go`:
```go
// Lines 200, 273, 296, 299, 316, 335, 346
fmt.Printf("Sport: %s, Position: %s, FantasyPositions: %v\n", sport, rosterSlot.Position, rosterSlot.FantasyPositions)
fmt.Printf("Debug - Golf generation, picked: %d/%d\n", pickedCount, totalSlots)
fmt.Printf("Generating golf lineup %d\n", len(lineups)+1)
// etc.
```

Should be replaced with structured logging:
```go
log.WithFields(log.Fields{
    "sport": sport,
    "position": rosterSlot.Position,
    "fantasy_positions": rosterSlot.FantasyPositions,
}).Debug("Processing roster slot")
```

## DOCUMENTATION:

- Go logging best practices: https://github.com/sirupsen/logrus
- Existing logging patterns in codebase (check other handlers)
- Log levels: Debug should only appear in development mode

## OTHER CONSIDERATIONS:

- Debug statements are currently polluting production logs
- No structured logging makes debugging harder
- Printf statements don't respect log levels
- Need to ensure sensitive data isn't logged (player salaries, user data)
- Should implement contextual logging with request IDs
- Performance impact: Printf is synchronous and can slow down optimization
- Consider adding performance metrics logging for optimization duration