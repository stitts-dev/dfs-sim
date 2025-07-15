# /create-prd - Create INIT PRD Files

Creates a new INIT Product Requirements Document based on the INITIAL_template.md template. This command focuses on planning and documentation rather than implementation.

## Usage

```
/create-prd [description or feature-name]
```

## Examples

```
/create-prd user-preferences-dashboard
/create-prd real-time-optimization-progress
/create-prd stripe-payment-integration
/create-prd drag-drop-lineup-builder
/create-prd fix phone input autofill handling
```

## What it does

1. Takes a feature description or name as input
2. Creates a new file in `INITs/` directory named `INITIAL_[feature-name].md`
3. Populates it with a comprehensive PRD using the INITIAL_template.md structure
4. Provides detailed analysis and planning without making code changes
5. Ready for review and approval before implementation begins

## Template Structure

The generated INIT file includes all standard PRD sections:

### Core Sections
- **FEATURE**: Brief feature description
- **CONTEXT & MOTIVATION**: Problem statement and value proposition
- **EXAMPLES**: Reference examples from `examples/` folder
- **CURRENT STATE ANALYSIS**: Existing components and constraints

### Technical Requirements
- **Backend Requirements**: API endpoints, database schema, business logic
- **Frontend Requirements**: UI components, state management, user flows
- **Infrastructure Requirements**: Environment variables, Docker, caching, security

### Implementation Planning
- **Phase 1**: Foundation (core components)
- **Phase 2**: Integration (data flow connections)
- **Phase 3**: Enhancement (advanced features, optimization)

### Quality Assurance
- **Testing Strategy**: Unit, integration, and E2E test plans
- **Success Criteria**: Definition of done
- **Monitoring & Observability**: Logging, metrics, alerts
- **Rollback Plan**: Safe deployment reversal strategy

### Risk Management
- **Potential Challenges & Risks**: Technical hurdles and dependencies
- **Other Considerations**: Project-specific gotchas and requirements

## Planning-First Approach

This command emphasizes **planning before implementation**:
- Creates comprehensive analysis without making code changes
- Provides detailed technical requirements and risk assessment
- Includes implementation phases and testing strategies
- Allows for review and approval before development begins
- Reduces technical debt and implementation risks through thorough planning

## File Naming Convention

Files are created as: `INITs/INITIAL_[feature-name].md`

Examples:
- `/create-prd payment-processing` → `INITs/INITIAL_payment-processing.md`
- `/create-prd websocket-hub` → `INITs/INITIAL_websocket-hub.md`

## Integration with DFS Project

This command integrates with the DFS Lineup Optimizer project structure:
- References microservices architecture (`services/`)
- Considers Supabase database integration
- Includes React + TypeScript frontend requirements
- Accounts for Docker deployment and Redis caching
