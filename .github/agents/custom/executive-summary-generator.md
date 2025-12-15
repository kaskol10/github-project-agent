# Agent: Executive Summary Generator

**Type**: custom

**Purpose**: Generate high-level summaries and dashboards for executive stakeholders (CEO, CTO, CFO) to quickly understand project status, risks, and strategic insights.

## Trigger

- schedule: "0 9 * * 1"  # Every Monday at 9 AM UTC
- schedule: "0 17 * * 5"  # Every Friday at 5 PM UTC
- manual: true

## Guidelines

- Focus on strategic insights, not technical details
- Highlight business value and impact
- Identify risks and opportunities
- Use executive-friendly language
- Include key metrics and trends
- Keep summaries concise (1-2 pages max)

## Actions

1. Collect all project issues and calculate metrics (aggregate issues by status)
2. Generate executive summary using LLM (call LLM with prompt template)
3. Create summary issue with executive insights (create issue with summary)

## Configuration

```yaml
summary_frequency: "weekly"
include_metrics: true
include_risks: true
include_trends: true
target_audience: "executive"
summary_format: "dashboard"
post_to_channel: false
create_summary_issue: true
```

## Prompt Template

- path: `prompts/executive-summary.md`
- fallback: hardcoded prompt

