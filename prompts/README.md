# Agent Prompts

This directory contains markdown prompt templates for all agents. These templates use Go template syntax for variable substitution.

## Template Syntax

Templates support:
- `{{.VariableName}}` - Simple variable substitution
- `{{if .Condition}}...{{end}}` - Conditional blocks
- `{{range .Items}}...{{end}}` - Loops

## Available Prompts

- `roaster.md` - Product analysis and roadmap suggestions
- `validator.md` - Task format validation and fixing
- `monitor.md` - Stale task status check messages

## Adding New Prompts

1. Create a new `.md` file in this directory
2. Use Go template syntax for variables
3. Reference it in your agent code using the `prompts` package

## Example

```markdown
# My Agent Prompt

Hello {{.UserName}}!

{{if .HasIssues}}
You have {{.IssueCount}} issues to address.
{{end}}
```

