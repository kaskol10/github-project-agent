# Agent: Progress Reporter

**Purpose**: Generate automated progress reports for Project Managers to share with stakeholders, tracking progress against milestones and identifying risks.

## Trigger

- schedule: "0 9 * * 1"  # Every Monday at 9 AM UTC
- schedule: "0 17 * * 5"  # Every Friday at 5 PM UTC
- manual: true

## Guidelines

- Track progress metrics: completion rate, velocity, burndown
- Identify risks and blockers
- Compare against milestones and deadlines
- Highlight achievements and wins
- Use clear, actionable language

## Actions

1. Collect all project issues and calculate metrics (aggregate issues by status)
2. Generate progress report using LLM (call LLM with prompt template)
3. Create report issue with progress summary (create issue with report)

## Configuration

```yaml
report_frequency: "weekly"
include_metrics: true
include_risks: true
include_milestones: true
create_report_issue: true
post_to_channel: false
target_audience: "stakeholders"
```

## Prompt Template

- path: `prompts/progress-reporter.md`
- fallback: hardcoded prompt

