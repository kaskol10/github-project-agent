# Agent: Task Validator

**Type**: core

**Purpose**: Enforce MCP task validation schema and automatically fix formatting issues according to the structured task format.

## Trigger

- event: issues.opened
- event: issues.edited
- schedule: "0 9 * * *"  # Daily at 9 AM UTC
- manual: true

## Validation Schema

The validator enforces the MCP Task Validation Schema defined in `.github/agents/core/mcp-task-validator.md`, which includes:

### Task Metadata (Required)
- **Priority**: Must be one of [Critical | High | Medium | Low]
- **Complexity**: Must be one of [Complex | Moderate | Trivial]
- **Days** (optional): Positive integer
- **Start Date**: Valid date in YYYY-MM-DD format, not in the past
- **End Date**: Valid date in YYYY-MM-DD format, must be ≥ Start Date

### Task Content Structure (Required)
1. **Description**: Must explain "what" and "why" with business value, dependencies, and blockers
2. **Scope**: Must define what's IN scope and explicitly OUT of scope
3. **Acceptance Criteria**: Must be specific, measurable, testable, using checkbox format `- [ ]`
4. **Definition of Done** (optional): Non-functional requirements checklist
5. **Additional Notes** (optional): Implementation details, technical specs, references

### Validation Rules
- Acceptance Criteria must be specific and measurable (no vague terms like "good", "nice", "better")
- Criteria should be testable/verifiable
- Criteria should not be interdependent
- Each criterion should use checkbox format `- [ ]`
- Avoid "and" within a single criterion (split into multiple)

## Actions

1. If a specific issue is provided:
   - Check if it has `agent-validator` label
   - If label exists: Issue is already validated (but continue to check all other issues)
   - If label doesn't exist: Issue needs validation (validate all issues)
2. Get all open issues in the project
3. Filter issues that don't have the `agent-validator` label
4. For each unvalidated issue:
   - Load MCP validation schema from `.github/agents/core/mcp-task-validator.md`
   - Check task format compliance against MCP schema:
     - Validate metadata (Priority, Complexity, Dates)
     - Validate content structure (Description, Scope, Acceptance Criteria)
     - Check for vague language in criteria
     - Verify dates are valid and End Date ≥ Start Date
   - If violations found:
     - Use LLM to fix formatting according to MCP schema
     - Preserve original content in a collapsible section
     - Add fixed content with agent modification notice
     - Update issue body (original content is preserved)
     - Add comment explaining fixes and suggestions
   - Add `agent-validator` label to mark issue as validated
5. Report validation results (summary of all validated issues)

**Behavior**:
- **If issue doesn't pass validation** (doesn't have `agent-validator` label): All unvalidated issues in the project are analyzed
- **If no specific issue provided**: All unvalidated issues in the project are analyzed
- **If all issues already validated**: Returns summary indicating all issues are validated

**Note**: 
- The agent preserves the original issue description in a collapsible section at the bottom, ensuring no information is lost.
- The `agent-validator` label prevents repeated validation runs on the same issue, avoiding unnecessary processing and comments.
- When validating, the agent processes ALL unvalidated issues in the project, not just the one specified (if any).
- The validator follows the MCP Task Validation Schema for consistency and quality.

## Configuration

```yaml
validation_schema_path: .github/agents/core/mcp-task-validator.md
guidelines_path: .github/task-guidelines.md  # Optional: additional project-specific guidelines
auto_fix: true
comment_on_fix: true
require_metadata: true
require_scope: true
require_acceptance_criteria: true
```

## Prompt Template

- path: `prompts/validator.md`
- fallback: hardcoded prompt with MCP schema validation rules

