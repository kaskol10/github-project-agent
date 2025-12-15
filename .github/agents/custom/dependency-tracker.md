# Agent: Dependency Tracker

**Type**: custom

**Purpose**: Track and visualize task dependencies to help Product Owners and Project Managers identify blockers, plan sprints, and manage critical paths.

## Trigger

- event: issues.opened
- event: issues.edited
- schedule: "0 8 * * *"  # Daily at 8 AM UTC
- manual: true

## Guidelines

- Parse task descriptions for dependency mentions ("depends on", "blocks", "requires", "needs")
- Build dependency graph
- Identify circular dependencies
- Alert on blocked tasks
- Track critical path
- Visualize dependencies in comments

## Actions

1. Extract dependencies and blockers from task description (analyze task content)
2. Generate dependency analysis using LLM (call LLM with prompt template)
3. Add comment with dependency analysis (add comment with generated content)

## Configuration

```yaml
dependency_keywords:
  - "depends on"
  - "blocks"
  - "requires"
  - "needs"
  - "waiting for"
alert_on_blockers: true
visualize_graph: true
track_critical_path: true
auto_link_issues: true
```

## Prompt Template

- path: `prompts/dependency-tracker.md`
- fallback: hardcoded prompt

