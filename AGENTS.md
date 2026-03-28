# Agentic Coding Guidelines for Rhino

This document provides guidelines for working with agentic coding systems on the Rhino workflow automation project.

## General Principles

### 1. Clear Task Definition
- Always start with a clear, specific task definition
- Break complex tasks into smaller, manageable subtasks
- Use the todo system to track progress

### 2. Code Understanding First
- Read and understand existing code before making changes
- Use `grep` to find relevant code patterns
- Examine the directory structure to understand the project layout

### 3. Minimal Changes
- Make the smallest change necessary to achieve the goal
- Follow existing code patterns and conventions
- Preserve existing functionality unless explicitly asked to change it

## Workflow for Agentic Coding

### Phase 1: Orient
1. **Understand the Goal**: Restate the objective in one line
2. **Determine Task Type**: Investigate (analysis) or Change (modification)
3. **Explore the Codebase**:
   - Use `read_file` to examine relevant files
   - Use `grep` to search for patterns
   - Understand dependencies and conventions

### Phase 2: Plan (for Change tasks)
1. **List Files to Change**: Identify exactly which files need modification
2. **Specific Changes**: Define what change will be made in each file
3. **Multi-file Changes**: Use a numbered checklist
4. **Single-file Fix**: One-line plan

### Phase 3: Execute & Verify
1. **Apply Changes**: Use `search_replace` for targeted modifications
2. **Verify Changes**:
   - Read back modified files to confirm changes
   - Run tests if available
   - Build the project to ensure no compilation errors
3. **Iterate if Needed**: If verification fails, re-examine and adjust

## Best Practices

### Error Handling
- Use descriptive error messages with context
- Include relevant variable values in error messages
- Use error wrapping (`%w`) for error chaining
- Validate inputs early and fail fast

### Logging
- Use structured logging with context fields
- Include relevant identifiers (task names, workflow names)
- Differentiate between info, error, and debug logs
- Avoid logging sensitive information

### Validation
- Validate configurations at startup
- Validate task parameters before execution
- Provide clear error messages for validation failures
- Use appropriate validation libraries (e.g., cron parsing, URL parsing)

### Code Quality
- Follow Go conventions and idioms
- Use consistent naming (camelCase for variables, PascalCase for types)
- Add comments for complex logic
- Keep functions focused and small

## Tools and When to Use Them

### `read_file`
- Read entire files or specific sections
- Use `offset` and `limit` for large files
- Always read before modifying

### `grep`
- Search for patterns across the codebase
- Find function definitions and usages
- Locate error messages and logging statements

### `search_replace`
- Make targeted changes to files
- Use exact text matching
- Apply one logical change at a time
- Verify changes after application

### `bash`
- Run build commands
- Execute tests
- Check project status
- Avoid using for file operations (use dedicated tools)

### `todo`
- Track task progress
- Break down complex tasks
- Mark tasks as in_progress when starting
- Mark as completed only when verified

## Example Workflow

### Task: Improve Error Handling

1. **Orient**:
   ```
   Goal: Improve error handling in workflow validation
   Task Type: Change
   ```

2. **Explore**:
   - Read `internal/models/workflow.go` to understand current validation
   - Search for error patterns with `grep`
   - Examine provider validation code

3. **Plan**:
   ```
   Files to change:
   1. internal/models/workflow.go - Enhance Validate() method
   2. internal/providers/http_provider.go - Improve parameter validation
   3. internal/providers/shell_provider.go - Add type checking
   ```

4. **Execute**:
   - Apply changes using `search_replace`
   - Verify each change by reading back the file
   - Build the project to ensure no errors

5. **Verify**:
   - Run tests if available
   - Manually test error scenarios
   - Confirm all changes work as expected

## Common Pitfalls to Avoid

1. **Over-engineering**: Don't add features not requested
2. **Breaking Changes**: Don't remove functionality without explicit instruction
3. **Inconsistent Style**: Follow existing code patterns
4. **Unverified Changes**: Always verify changes work
5. **Scope Creep**: Stick to the defined task

## Collaboration

When working with human developers:
- Ask clarifying questions if the task is unclear
- Provide progress updates
- Explain technical decisions when asked
- Suggest improvements but implement only what's requested

## Security Considerations

- Fix security vulnerabilities immediately if discovered
- Don't log sensitive information
- Validate all external inputs
- Use secure coding practices

By following these guidelines, agentic coding systems can effectively contribute to the Rhino project while maintaining code quality and consistency.