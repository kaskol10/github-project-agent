# Agent: Task Summarizer

**Type**: custom

**Purpose**: Generate concise summaries of tasks and issues to help team members quickly understand key information.

## Trigger

- event: issues.opened
- event: issues.edited
- schedule: "0 10 * * *"  # Daily at 10 AM UTC
- manual: true

## Guidelines

- Generate summaries for tasks longer than 200 characters
- Focus on: objective, key requirements, and dependencies
- Keep summaries under 100 words
- Include priority and status if available

## Actions

1. Check if task body is longer than threshold (minimum length check)
2. Generate concise summary using LLM (call LLM with prompt template)
3. Add summary as a comment on the issue (add comment with generated content)

## Configuration

```yaml
min_length_for_summary: 50
max_summary_length: 100
include_priority: true
include_status: true
add_label: "summarized"
summary_format: "bullet-points"
```

## Prompt Template

- path: `prompts/summarizer.md`
- fallback: hardcoded prompt

