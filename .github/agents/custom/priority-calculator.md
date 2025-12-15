# Agent: Priority Calculator

**Type**: custom

**Purpose**: Automatically calculate and suggest task priorities based on business value, effort, dependencies, and strategic alignment.

## Trigger

- event: issues.opened
- event: issues.edited
- condition: labels.contains("needs-priority")
- manual: true

## Guidelines

- Consider business impact and user value
- Factor in technical complexity and effort
- Account for dependencies and blockers
- Align with product strategy and roadmap
- Use standard priority labels: P0 (Critical), P1 (High), P2 (Medium), P3 (Low)

## Actions

1. Analyze task for priority factors (call LLM with prompt template)
2. Calculate priority score and suggest label
3. Add comment with priority assessment (add comment with generated content)

## Configuration

```yaml
priority_labels:
  - "priority:P0"
  - "priority:P1"
  - "priority:P2"
  - "priority:P3"
auto_apply: false  # Suggest only, don't auto-apply
weight_business_value: 0.4
weight_effort: 0.2
weight_dependencies: 0.2
weight_strategic_alignment: 0.2
min_confidence: 0.7  # Only suggest if confidence > 70%
```

## Prompt Template

- path: `prompts/priority-calculator.md`
- fallback: hardcoded prompt

