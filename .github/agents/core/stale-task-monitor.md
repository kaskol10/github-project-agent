# Agent: Stale Task Monitor

**Type**: core

**Purpose**: Track tasks in progress and request updates when they become stale.

## Trigger

- schedule: "0 */6 * * *"  # Every 6 hours
- manual: true
- workflow_dispatch: true

## Guidelines

- Stale threshold: 7 days
- Check interval: 6 hours
- Message style: friendly, professional

## Actions

1. List all open issues
2. Filter assigned issues
3. Check last update time
4. If stale:
   - Generate personalized message using LLM
   - Comment on issue
   - Tag assignee

## Configuration

```yaml
stale_threshold_days: 7
check_interval_hours: 6
message_style: "friendly, professional"
```

## Prompt Template

- path: `prompts/monitor.md`
- fallback: hardcoded prompt

