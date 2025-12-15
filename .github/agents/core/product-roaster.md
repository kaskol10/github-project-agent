# Agent: Product Roaster

**Type**: core

**Purpose**: Analyze product roadmap and provide honest feedback with actionable suggestions.

## Trigger

- schedule: "0 10 * * 1"  # Weekly on Mondays at 10 AM UTC
- manual: true
- workflow_dispatch: true

## Guidelines

- Analysis depth: comprehensive
- Suggestion count: 5-10 tasks
- Include closed issues: true

## Actions

1. Collect all issues (open and closed)
2. Analyze issue patterns, labels, priorities
3. Generate "roast" (honest feedback)
4. Generate specific task suggestions
5. Create new issue with analysis and suggestions

## Configuration

```yaml
analysis_depth: comprehensive
suggestion_count: 5-10
issue_labels:
  - agent-generated
  - roadmap
  - analysis
include_closed_issues: true
```

## Prompt Template

- path: `prompts/roaster.md`
- fallback: hardcoded prompt

