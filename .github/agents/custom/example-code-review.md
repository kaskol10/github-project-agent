# Agent: Code Review Enforcer

**Type**: custom

**Purpose**: Ensure pull requests meet review requirements before merging.

## Trigger

- event: pull_request.opened
- event: pull_request.synchronize
- condition: labels.contains("needs-review")
- schedule: "0 9 * * *"  # Daily at 9 AM UTC

## Guidelines

- Minimum approvals: 2
- Require passing tests: true
- Minimum description length: 100 characters
- Require linked issue: true

## Actions

1. Check if PR has at least 2 approvals
2. Verify all CI checks are passing
3. Check if description meets minimum length
4. Verify PR is linked to an issue
5. If all checks pass:
   - Add label: "ready-to-merge"
   - Comment: "✅ This PR is ready to merge!"
6. If checks fail:
   - Comment with list of missing requirements
   - Add label: "needs-attention"

## Configuration

```yaml
min_approvals: 2
require_passing_tests: true
min_description_length: 100
require_linked_issue: true
success_label: "ready-to-merge"
failure_label: "needs-attention"
```

## Prompt Template

```markdown
# Code Review Check Prompt

Check if this pull request meets all requirements:

- Title: {{.Title}}
- Description: {{.Description}}
- Approvals: {{.ApprovalCount}}/{{.MinApprovals}}
- CI Status: {{.CIStatus}}
- Linked Issues: {{.LinkedIssues}}

Provide a summary of what's missing or confirm it's ready.
```

