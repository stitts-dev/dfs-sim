# Execute PRP

Implement a feature using the PRP file.

## PRP File: $ARGUMENTS

## Execution Process

### 1. **Pre-execution Setup**
   - Mark PRP as in-progress (add execution timestamp to PRP metadata)
   - Confirm initial state and requirements

### 2. **Load PRP & Deep Analysis**
   - Read the specified PRP file thoroughly
   - Understand all context, requirements, and success criteria
   - Follow all instructions in the PRP and extend research if needed
   - Ensure you have all needed context to implement the PRP fully
   - Do more web searches and codebase exploration as needed
   - Identify dependencies and integration points with existing code

### 3. **ULTRATHINK & Planning**
   - Think hard before executing the plan. Create a comprehensive implementation strategy
   - Break down complex tasks into smaller, manageable steps using TodoWrite tool
   - Use the TodoWrite tool to create and track your implementation plan
   - Identify implementation patterns from existing code to follow
   - Plan validation strategy and testing approach
   - Consider rollback scenarios and risk mitigation

### 4. **Execute the Plan**
   - Execute the PRP systematically following the planned approach
   - Implement all code changes with high quality and attention to detail
   - Handle errors gracefully and adapt plan as needed

### 5. **Comprehensive Validation**
   - Run each validation command specified in the PRP
   - Execute full test suite: unit tests, integration tests, linting
   - Fix any failures immediately and re-run until all pass
   - Test edge cases and error scenarios
   - Validate all PRP requirements and success criteria are met
   - Double-check PRP compliance by re-reading requirements

### 6. **PRP Lifecycle Management**
   - Create executed PRPs directory if it doesn't exist: `mkdir -p @/executed`
   - Move PRP file to executed folder: `mv [prp-path] @/executed/`
   - If there's an associated PRD file, move it to `@/old/` directory
   - Create execution metadata file `@/executed/[prp-name].meta.json`:
     ```json
     {
       "prp_name": "[prp-title]",
       "executed_at": "[timestamp]",
       "executor": "claude-code",
       "status": "completed",
       "validation_status": "all_passed"
     }
     ```

### 7. **Completion Report & Handoff**
   - Provide comprehensive completion report including:
     - ✅ PRP execution status and all requirements met
     - 📊 Key metrics achieved vs. PRP success criteria
     - 🧪 Validation results summary (tests, linting, manual verification)
     - 📁 Files changed and major code areas affected
     - ⚡ Performance improvements or notable technical achievements
     - 🔄 Next steps for further development or integration
   - Confirm all TodoWrite items are marked complete
   - Re-read PRP one final time to ensure 100% compliance
   - Provide clear handoff for human reviewer

### 8. **Reference & Support**
   - PRP file moved to `@/executed/` for future reference
   - You can always reference the executed PRP or original requirements


## Error Handling & Recovery

- If validation fails, fix issues and re-run validation
- Use error patterns in PRP to guide troubleshooting approach
- If major issues arise, document and request guidance

## Success Criteria

- ✅ All PRP requirements implemented and validated
- ✅ All tests passing and validation complete
- ✅ PRP moved to executed status with metadata
- ✅ Ready for further development or integration