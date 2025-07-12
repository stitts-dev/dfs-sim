# Execute PRP with Git Workflow

Implement a feature using the PRP file with proper git workflow, branch management, and PR creation.

## PRP File: $ARGUMENTS

## Enhanced Execution Process

### 1. **Pre-execution Setup & Branch Creation**
   - Parse PRP filename to generate meaningful branch name (e.g., `prp/ai-recommendations-fix`)
   - Ensure we're on main branch and it's up to date: `git checkout main && git pull origin main`
   - Create and checkout new feature branch: `git checkout -b prp/[feature-name]`
   - Mark PRP as in-progress (add execution timestamp and branch info to PRP metadata)
   - Confirm branch creation and initial state

### 2. **Load PRP & Deep Analysis**
   - Read the specified PRP file thoroughly
   - Understand all context, requirements, and success criteria
   - Follow all instructions in the PRP and extend research if needed
   - Ensure you have all needed context to implement the PRP fully
   - Do more web searches and codebase exploration as needed
   - Identify dependencies and integration points with existing code

### 3. **ULTRATHINK & Git Strategy Planning**
   - Think hard before executing the plan. Create a comprehensive implementation strategy
   - Break down complex tasks into smaller, manageable steps using TodoWrite tool
   - Plan commit strategy: identify logical commit boundaries for incremental progress
   - Use the TodoWrite tool to create and track your implementation plan
   - Identify implementation patterns from existing code to follow
   - Plan validation strategy and testing approach
   - Consider rollback scenarios and risk mitigation

### 4. **Execute the Plan with Incremental Commits**
   - Execute the PRP systematically following the planned approach
   - Implement all code changes with high quality and attention to detail
   - Make incremental commits at logical milestones to maintain clean git history
   - Use descriptive commit messages following project conventions
   - Ensure each commit represents a working state when possible
   - Handle errors gracefully and adapt plan as needed

### 5. **Comprehensive Validation**
   - Run each validation command specified in the PRP
   - Execute full test suite: unit tests, integration tests, linting
   - Fix any failures immediately and re-run until all pass
   - Verify no uncommitted changes: `git status` should be clean
   - Test edge cases and error scenarios
   - Validate all PRP requirements and success criteria are met
   - Double-check PRP compliance by re-reading requirements

### 6. **Final Commit & Branch Preparation**
   - Stage all final changes: `git add .`
   - Create comprehensive final commit with PRP completion message
   - Use commit message format: `feat: implement [PRP-TITLE] - [brief description]`
   - Include reference to PRP file and key improvements in commit body
   - Verify git log shows clean, logical progression of changes
   - Ensure branch is ready for review and merge

### 7. **Push & Pull Request Creation**
   - Push feature branch to remote: `git push -u origin prp/[feature-name]`
   - Create pull request using GitHub CLI with comprehensive description:
     ```bash
     gh pr create --title "PRP: [PRP-TITLE]" --body "$(cat <<'EOF'
     ## PRP Implementation: [PRP-TITLE]
     
     **PRP File**: [path-to-prp]
     **Branch**: prp/[feature-name]
     
     ## Summary
     [3-5 bullet points of key changes and improvements]
     
     ## Implementation Details
     [Brief technical overview of approach taken]
     
     ## Testing & Validation
     - [ ] All unit tests passing
     - [ ] Integration tests passing  
     - [ ] Linting and code quality checks passed
     - [ ] PRP requirements fully implemented
     - [ ] Manual testing completed
     
     ## Success Metrics
     [List key metrics/criteria from PRP that were achieved]
     
     ## Files Changed
     [Brief overview of major files modified/added]
     
     🤖 Generated with [Claude Code](https://claude.ai/code)
     
     Closes: [PRP-file-reference]
     EOF
     )"
     ```

### 8. **PRP Lifecycle Management**
   - Create executed PRPs directory if it doesn't exist: `mkdir -p PRPs/executed`
   - Move PRP file to executed folder: `mv [prp-path] PRPs/executed/`
   - Create execution metadata file `PRPs/executed/[prp-name].meta.json`:
     ```json
     {
       "prp_name": "[prp-title]",
       "executed_at": "[timestamp]",
       "branch": "prp/[feature-name]",
       "pr_url": "[github-pr-url]",
       "executor": "claude-code",
       "status": "completed",
       "validation_status": "all_passed"
     }
     ```
   - Commit PRP lifecycle changes: `git add PRPs/ && git commit -m "docs: mark [PRP-TITLE] as executed"`
   - Push lifecycle changes: `git push`

### 9. **Completion Report & Handoff**
   - Provide comprehensive completion report including:
     - ✅ PRP execution status and all requirements met
     - 🔗 Pull Request URL for review
     - 📊 Key metrics achieved vs. PRP success criteria
     - 🧪 Validation results summary (tests, linting, manual verification)
     - 📁 Files changed and major code areas affected
     - ⚡ Performance improvements or notable technical achievements
     - 🔄 Next steps for code review and merge process
   - Confirm all TodoWrite items are marked complete
   - Re-read PRP one final time to ensure 100% compliance
   - Provide clear handoff for human reviewer

### 10. **Reference & Support**
   - PRP file moved to `PRPs/executed/` for future reference
   - Branch remains available for continued development if needed
   - All implementation details documented in PR description
   - You can always reference the executed PRP or original requirements

## Git Workflow Requirements

- **Branch Naming**: Use `prp/[descriptive-kebab-case-name]` format
- **Commit Messages**: Follow conventional commits (feat:, fix:, docs:, etc.)
- **PR Title**: Use "PRP: [Original PRP Title]" format
- **Clean History**: Logical, incremental commits with clear progression
- **Testing**: All tests must pass before PR creation
- **Documentation**: PR description must be comprehensive and self-explanatory

## Error Handling & Recovery

- If validation fails, fix issues on the same branch and re-run validation
- Use error patterns in PRP to guide troubleshooting approach
- If major issues arise, document in PR and request guidance
- Maintain clean git history even during error recovery
- Never force push or rewrite shared branch history

## Success Criteria

- ✅ All PRP requirements implemented and validated
- ✅ Clean git history with meaningful commits
- ✅ Comprehensive PR created with detailed description
- ✅ All tests passing and validation complete
- ✅ PRP moved to executed status with metadata
- ✅ Ready for human code review and merge