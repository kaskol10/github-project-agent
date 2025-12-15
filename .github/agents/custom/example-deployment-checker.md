# Agent: Deployment Checker

**Type**: custom

**Purpose**: Monitor deployment status and notify team of issues.

## Trigger

- event: deployment.created
- event: deployment_status
- schedule: "0 */4 * * *"  # Every 4 hours

## Guidelines

- Check deployment status
- Monitor for failed deployments
- Notify on rollback

## Actions

1. Check deployment status
2. If failed:
   - Create issue with deployment details
   - Tag deployment team
   - Add label: "deployment-failed"
3. If successful:
   - Comment on related PR
   - Update deployment tracking

## Configuration

```yaml
notify_on_failure: true
create_issue_on_failure: true
tag_team: "deployment-team"
failure_label: "deployment-failed"
```

## Prompt Template

```markdown
# Deployment Status Prompt

Analyze this deployment:

- Environment: {{.Environment}}
- Status: {{.Status}}
- Commit: {{.CommitSHA}}
- Duration: {{.Duration}}

Provide a status update and any recommendations.
```

