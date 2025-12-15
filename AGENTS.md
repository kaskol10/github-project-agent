# Project Agents Guide

This file provides comprehensive guidance for AI assistants and coding agents (like Claude, Gemini, Cursor, and others) to work with this codebase effectively and maintain consistency.

## Project Overview

This repository contains the **GitHub Project Agent**, a Go-based intelligent agent system for managing GitHub project tasks with automated validation, monitoring, and product analysis capabilities. The agent uses self-hosted LLMs (via litellm/vllm) and integrates with GitHub's API to provide intelligent task management across single repositories or GitHub Projects spanning multiple repositories.

## Project Structure and Repository Layout

The project follows standard Go package conventions:

```
github-project-agent/
├── main.go                 # Main application entry point with CLI interface
├── agent/                  # Core agent implementations
│   ├── validator.go        # Task format validation and auto-fixing
│   ├── monitor.go          # Stale task detection and notifications
│   ├── roaster.go          # Product analysis and roadmap suggestions
│   └── prompts_helper.go   # Shared helper for prompt loading
├── config/                 # Configuration management
│   └── config.go           # Loads configuration from environment variables
├── github/                 # GitHub API integration
│   ├── client.go           # Repository-specific client
│   ├── project_client.go   # GitHub Projects (multi-repo) client
│   └── unified_client.go   # Unified interface for both modes
├── guidelines/             # Guidelines parser
│   └── parser.go           # Parses markdown guidelines files
├── llm/                    # LLM client
│   └── client.go           # LiteLLM/VLLM HTTP client
├── mcp/                    # Model Context Protocol interface
│   └── interface.go       # MCP-compatible API for agent execution
├── plugins/                # Plugin system
│   ├── loader.go           # Loads plugin agents from .md files
│   └── executor.go        # Executes plugin-based agents
├── prompts/                # Markdown prompt templates
│   ├── loader.go           # Loads and renders prompt templates
│   └── *.md                # Prompt templates for each agent
├── .github/
│   ├── agents/             # Plugin agent definitions
│   │   ├── core/           # Core system agents
│   │   └── custom/         # User-defined custom agents
│   └── workflows/          # GitHub Actions workflows
└── *.md                    # Documentation files
```

## Coding Style and Standards

### Go Conventions

- **Package naming**: Use lowercase, single-word package names
- **File naming**: Use snake_case for multi-word files (e.g., `unified_client.go`)
- **Function naming**: Use PascalCase for exported functions, camelCase for private
- **Error handling**: Always return errors, never panic in library code
- **Context**: Always pass `context.Context` as the first parameter
- **Interfaces**: Define interfaces close to where they're used, not in separate files

### Code Organization

- **One package per directory**: Each directory is a single package
- **Interface segregation**: Keep interfaces small and focused
- **Dependency injection**: Use constructor functions that accept dependencies
- **No circular dependencies**: Package structure should be acyclic

### Example Code Pattern

```go
// Package-level documentation
package github

import (
	"context"
	"fmt"
)

// Interface definition
type UnifiedClient interface {
	ListIssues(ctx context.Context, state string) ([]*Issue, error)
	GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error)
}

// Implementation
type Client struct {
	token string
	owner string
	repo  string
}

// Constructor
func NewClient(token, owner, repo, baseURL string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("token required")
	}
	return &Client{
		token: token,
		owner: owner,
		repo:  repo,
	}, nil
}

// Method with proper error handling
func (c *Client) ListIssues(ctx context.Context, state string) ([]*Issue, error) {
	// Implementation
	return nil, nil
}
```

## Testing Patterns and Guidelines

### Test File Naming

- Test files must end with `_test.go`
- Test files belong to the same package (use `package github` not `package github_test`)
- For integration tests, use build tags: `//go:build integration`

### Test Structure

```go
package github

import (
	"context"
	"testing"
)

func TestClient_ListIssues(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		wantErr bool
	}{
		{
			name:    "valid state",
			state:   "open",
			wantErr: false,
		},
		{
			name:    "invalid state",
			state:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient("token", "owner", "repo", "")
			_, err := client.ListIssues(context.Background(), tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListIssues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

### Testing Requirements

- **Unit tests**: Test all exported functions and methods
- **Table-driven tests**: Use table-driven tests for multiple scenarios
- **Error cases**: Always test error paths
- **Integration tests**: Use build tags and separate files for integration tests
- **Mocking**: Use interfaces to enable easy mocking
- **Coverage**: Aim for >80% code coverage

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...

# Run integration tests
go test -tags=integration ./...
```

## LangChainGo Integration

### When to Use LangChainGo

- **Complex agent workflows**: Multi-step reasoning chains
- **Tool calling**: When agents need to call external APIs or functions
- **Memory management**: Conversation history and context
- **Structured outputs**: When you need structured data from LLM responses

### LangChainGo Patterns

```go
import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/agents"
)

// Example: Creating a chain
func createValidationChain(llm llms.Model) chains.Chain {
	return chains.NewLLMChain(llm, chains.WithPrompt(validationPrompt))
}

// Example: Agent with tools
func createAgentWithTools(llm llms.Model, tools []agents.Tool) *agents.Executor {
	return agents.NewExecutor(
		agents.NewConversationalAgent(llm, tools),
		tools,
	)
}
```

### Integration Guidelines

- **Keep it optional**: LangChainGo should enhance, not replace existing functionality
- **Progressive enhancement**: Start with simple LLM calls, add chains when needed
- **Tool definitions**: Define tools as Go functions with clear signatures
- **Error handling**: Wrap LangChainGo errors with context

## Local Execution

### Development Setup

1. **Install dependencies**:
   ```bash
   go mod download
   ```

2. **Set environment variables**:
   ```bash
   export GITHUB_TOKEN="your_token"
   export GITHUB_OWNER="your_org"
   export GITHUB_REPO="your_repo"
   export LITELLM_BASE_URL="http://localhost:4000"
   export LLM_MODEL="gpt-4"
   ```

3. **Run locally**:
   ```bash
   go run main.go -mode=validate
   ```

### Local Testing

- Use a test repository for development
- Mock GitHub API responses when possible
- Use local LLM servers (Ollama, vLLM) for faster iteration
- Enable verbose logging for debugging

## GitHub Actions Execution

### Workflow Structure

All workflows are in `.github/workflows/`:

- `agent-validate.yml` - Task validation on issue events
- `agent-monitor.yml` - Stale task monitoring
- `agent-roast.yml` - Product analysis

### Workflow Patterns

```yaml
name: Agent Task

on:
  issues:
    types: [opened, edited]
  schedule:
    - cron: '0 9 * * *'
  workflow_dispatch:
    inputs:
      issue_number:
        description: 'Issue number to validate'
        required: false

jobs:
  validate:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Run agent
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          LITELLM_BASE_URL: ${{ secrets.LITELLM_BASE_URL }}
          LLM_MODEL: ${{ secrets.LLM_MODEL }}
        run: |
          go run main.go -mode=validate -issue=${{ inputs.issue_number }}
```

### Workflow Best Practices

- **Self-hosted runners**: All workflows use self-hosted runners
- **Secrets management**: Use GitHub Secrets for sensitive data
- **Idempotency**: Workflows should be safe to run multiple times
- **Error handling**: Fail gracefully with clear error messages
- **Logging**: Use structured logging for debugging

## GitHub Actions Marketplace Integration

### Using Marketplace Actions

When integrating with GitHub Actions Marketplace:

1. **Pin versions**: Always pin to specific versions, not `@main`
2. **Verify actions**: Review action source code before using
3. **Minimal permissions**: Use least privilege principle
4. **Document dependencies**: List all marketplace actions in README

### Example Marketplace Integration

```yaml
- name: Setup Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.21'

- name: Checkout code
  uses: actions/checkout@v4

- name: Run tests
  uses: actionsx/golangci-lint@v1
  with:
    version: latest
```

### Distributed Execution Patterns

- **Matrix strategies**: Use for testing multiple Go versions
- **Job dependencies**: Use `needs` to create execution pipelines
- **Parallel execution**: Run independent jobs in parallel
- **Caching**: Cache Go modules and build artifacts

## Feature Development

### Adding New Features

1. **Create an issue**: Document the feature requirement
2. **Design the API**: Define interfaces and function signatures
3. **Write tests first**: Follow TDD when possible
4. **Implement**: Write the feature code
5. **Document**: Update README and relevant docs
6. **Test**: Run all tests and verify locally
7. **Review**: Submit PR with clear description

### Adding New Agents

Use the plugin system - see [PLUGINS.md](PLUGINS.md):

1. Create `.md` file in `.github/agents/custom/`
2. Define agent following the standard format
3. Add prompt template if needed
4. Test using MCP mode
5. Document in plugin file

### Code Review Checklist

- [ ] Code follows Go conventions
- [ ] Tests are included and passing
- [ ] Documentation is updated
- [ ] Error handling is comprehensive
- [ ] No hardcoded values (use config)
- [ ] Logging is appropriate
- [ ] No security issues (secrets, etc.)

## Linting and Code Quality

### Linting Tools

- **golangci-lint**: Primary linter (configure in `.golangci.yml`)
- **go vet**: Built-in static analysis
- **go fmt**: Code formatting

### Linting Configuration

Create `.golangci.yml`:

```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports

linters-settings:
  errcheck:
    check-type-assertions: true
  goimports:
    local-prefixes: github.com/yourusername/github-project-agent
```

### Pre-commit Hooks

```bash
#!/bin/sh
# .git/hooks/pre-commit

go fmt ./...
go vet ./...
golangci-lint run
```

### CI Integration

Add linting to GitHub Actions:

```yaml
- name: Run linters
  run: |
    go fmt ./...
    go vet ./...
    golangci-lint run
```

## Error Handling Patterns

### Error Wrapping

Always wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to list issues: %w", err)
}
```

### Error Types

Define custom error types when appropriate:

```go
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error in %s: %s", e.Field, e.Message)
}
```

### Error Logging

Log errors with context:

```go
log.Printf("Failed to validate issue %d: %v", issueNumber, err)
```

## Configuration Management

### Environment Variables

- Use environment variables for all configuration
- Provide sensible defaults
- Document all variables in README
- Validate required variables at startup

### Configuration Structure

```go
type Config struct {
	GitHub struct {
		Token    string
		Owner    string
		Repo     string
		ProjectID string
	}
	LLM struct {
		BaseURL string
		Model   string
		APIKey  string
	}
}
```

## Documentation Standards

### Code Comments

- **Package comments**: Describe package purpose
- **Exported functions**: Document parameters, return values, errors
- **Complex logic**: Explain why, not what
- **Examples**: Include usage examples for complex functions

### Markdown Documentation

- **README.md**: Project overview and quick start
- **PLUGINS.md**: Plugin system documentation
- **ADDING_AGENTS.md**: Guide for adding agents
- **AGENTS.md**: This file - project guidelines

### Documentation Format

```go
// ProcessIssue validates and fixes an issue according to guidelines.
// It returns true if the issue was valid, false if it was fixed.
// An error is returned if processing fails.
func ProcessIssue(ctx context.Context, issue *Issue) (bool, error) {
	// Implementation
}
```

## Security Best Practices

- **Never commit secrets**: Use environment variables or secrets management
- **Token rotation**: Support token rotation without code changes
- **Input validation**: Validate all user inputs
- **Rate limiting**: Respect API rate limits
- **Error messages**: Don't leak sensitive information in errors

## Performance Considerations

- **Context timeouts**: Always set timeouts for external calls
- **Connection pooling**: Reuse HTTP clients
- **Caching**: Cache expensive operations when appropriate
- **Parallel execution**: Use goroutines for independent operations
- **Resource cleanup**: Close connections and files properly

## Migration and Backward Compatibility

- **Versioning**: Use semantic versioning
- **Deprecation**: Mark deprecated features clearly
- **Migration guides**: Provide migration paths for breaking changes
- **Feature flags**: Use environment variables for feature toggles

## AI Assistant Guidelines

When working on this codebase:

1. **Follow Go conventions**: Use standard Go patterns and idioms
2. **Maintain consistency**: Match existing code style
3. **Add tests**: Include tests for new functionality
4. **Update docs**: Keep documentation current
5. **Use plugins**: Prefer plugin system over code changes for agents
6. **Respect interfaces**: Don't break existing interfaces
7. **Error handling**: Always handle errors properly
8. **Logging**: Use structured logging with context

## Questions to Ask

Before making changes:

- Does this follow existing patterns?
- Are tests included?
- Is documentation updated?
- Does it work in both repo and project modes?
- Is it compatible with the plugin system?
- Are errors handled properly?
- Is logging appropriate?

## References

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go)
- [LangChainGo Documentation](https://github.com/tmc/langchaingo)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

