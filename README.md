# GitHub Project Agent

An intelligent agent for managing GitHub project tasks with automated validation, monitoring, and product analysis capabilities.

> 🚀 **New to GitHub Actions?** Get started in [under 2 minutes](QUICKSTART.md)!

## Features

- **🤖 Task Format Enforcement**: Automatically validates and fixes task formatting to ensure consistency across your team
- **⏰ Stale Task Monitoring**: Tracks tasks in progress and automatically asks for updates when they've been stale for too long
- **🔥 Product Roasting**: Analyzes your product roadmap and provides honest feedback with actionable task suggestions

## Architecture

- Built with **Go** and **LangchainGo** (ready for integration)
- Uses **litellm/vllm** for self-hosted LLM inference
- Integrates with GitHub API for issue management
- **Dual Mode Support**: Works with single repositories or GitHub Projects spanning multiple repos
- **Markdown-based guidelines**: Define task format rules in `.md` files
- **Markdown-based prompts**: Customize agent prompts in `.md` files (no code changes needed!)
- **Plugin System**: Add custom agents without modifying code - just add `.md` files to `.github/agents/custom/`
- **MCP (Model Context Protocol) interface**: Standardized API for agent execution
- **GitHub Actions ready**: Run as self-hosted GitHub Actions workflows

## Prerequisites

- Go 1.21 or later
- **Authentication**: Either:
  - A GitHub Personal Access Token with `repo` scope (legacy), or
  - A GitHub App with `Issues: Read and write` permission (recommended)
- A running litellm or vllm server

> **Note**: GitHub App authentication is recommended for better security, automatic token refresh, and higher rate limits. See [GITHUB_APP_SETUP.md](GITHUB_APP_SETUP.md) for setup instructions.

## Setup

1. **Clone and install dependencies:**
   ```bash
   go mod download
   ```

2. **Set environment variables:**

   **Option A: GitHub App Authentication (Recommended)**
   
   ```bash
   # GitHub App credentials
   export GITHUB_APP_ID="123456"
   export GITHUB_APP_INSTALLATION_ID="789012"
   export GITHUB_APP_PRIVATE_KEY_PATH="/path/to/private-key.pem"
   # OR provide key directly:
   # export GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----..."
   
   # Repository/Project configuration
   export GITHUB_OWNER="your_org_or_username"
   export GITHUB_REPO="your_repo_name"  # For repo mode
   # OR for project mode (multiple repos):
   # export GITHUB_PROJECT_ID="123"
   # export GITHUB_REPOS="owner/repo1,owner/repo2,owner/repo3"
   # Note: GITHUB_REPO is NOT needed in project mode - system searches across all repos automatically!
   
   # LLM configuration
   export LITELLM_BASE_URL="http://localhost:4000"
   export LLM_MODEL="gpt-4"
   export LLM_API_KEY=""  # Optional
   ```
   
   See [GITHUB_APP_SETUP.md](GITHUB_APP_SETUP.md) for detailed setup instructions.

   **Option B: Personal Access Token (Legacy)**
   
   ```bash
   export GITHUB_TOKEN="your_github_token"
   export GITHUB_OWNER="your_org_or_username"
   export GITHUB_REPO="your_repo_name"  # For repo mode
   # OR for project mode:
   # export GITHUB_PROJECT_ID="123"
   # export GITHUB_REPOS="owner/repo1,owner/repo2,owner/repo3"
   export LITELLM_BASE_URL="http://localhost:4000"
   export LLM_MODEL="gpt-4"
   export LLM_API_KEY=""  # Optional
   ```

   See `PROJECT_MODE.md` for detailed information about project mode.

3. **Optional configuration:**
   ```bash
   export STALE_TASK_THRESHOLD_DAYS=7  # Days before a task is considered stale
   export CHECK_INTERVAL_HOURS=24      # How often to check (for daemon mode)
   export GUIDELINES_PATH=".github/task-guidelines.md"  # Path to guidelines file
   ```

4. **Create guidelines file (optional but recommended):**
   
   Create `.github/task-guidelines.md` in your repository with your task format rules. See `.github/task-guidelines.md` for an example.

5. **Create plugin agents (optional but recommended):**
   
   Create agent definitions in `.github/agents/custom/` following the format in [PLUGINS.md](PLUGINS.md). See the examples in `.github/agents/core/` for reference.

6. **Review project guidelines (recommended for contributors):**
   
   See [AGENTS.md](AGENTS.md) for coding standards, testing patterns, LangChainGo integration, and development guidelines.

7. **Customize prompts (optional but recommended):**
   
   Edit markdown files in the `prompts/` directory to customize agent behavior. See `prompts/README.md` and `ADDING_AGENTS.md` for details.

## Usage

### Validate Task Format

The Task Validator automatically validates **all unvalidated issues** in the project. Issues with the `agent-validator` label are skipped to avoid repetition.

Validate all unvalidated issues (recommended):
```bash
go run main.go -mode=mcp -agent="Task Validator"
```

Validate starting from a specific issue (will still validate all unvalidated issues):
```bash
go run main.go -mode=mcp -agent="Task Validator" -issue=123
```

**Note**: If an issue doesn't have the `agent-validator` label, the agent will validate **all unvalidated issues** in the project, not just the specified one. This ensures comprehensive validation across the entire project.

### Monitor Stale Tasks

Run once to check for stale tasks:
```bash
go run main.go -mode=monitor -once
```

Run as a daemon (continuously monitors):
```bash
go run main.go -mode=monitor -daemon
```

### Generate Product Roast & Suggestions

```bash
go run main.go -mode=roast
```

This will create a new GitHub issue with:
- Honest analysis of your product/roadmap
- Actionable task suggestions

### Run All Tasks

```bash
go run main.go -mode=all
```

### MCP Mode (Plugin Agents)

List available agents:
```bash
go run main.go -mode=mcp
```

Execute a specific agent:
```bash
go run main.go -mode=mcp -agent="Task Validator" -issue=123
```

See [PLUGINS.md](PLUGINS.md) for detailed information about the plugin system.

## Using as GitHub Action

You can use this agent as a GitHub Action in your workflows. **Get started in under 2 minutes!** 🚀

> 📖 **New to GitHub Actions?** See [QUICKSTART.md](QUICKSTART.md) for a step-by-step guide.

### Quick Start (Simplest Setup)

Create `.github/workflows/task-validator.yml`:

```yaml
name: Task Validator

on:
  issues:
    types: [opened, edited]

jobs:
  validate:
    runs-on: self-hosted
    permissions:
      issues: write
      contents: read
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Validate Tasks
        uses: your-org/github-project-agent@v1.0.0
        with:
          mode: 'mcp'
          agent: 'Task Validator'
          litellm_base_url: ${{ secrets.LITELLM_BASE_URL }}
          llm_model: ${{ secrets.LLM_MODEL || 'gpt-4' }}
```

**That's it!** The action automatically:
- ✅ Uses GitHub's built-in token (`secrets.GITHUB_TOKEN`) - no setup needed!
- ✅ Detects your repository from workflow context - no configuration needed!
- ✅ Validates all issues automatically

**Only 1 secret required**: `LITELLM_BASE_URL` (your LLM server URL)

See [QUICKSTART.md](QUICKSTART.md) for complete setup instructions.

### Testing Without Publishing

You can test the action in GitHub Actions **without publishing to the Marketplace**:

1. **Test in same repository** (easiest):
   ```yaml
   - uses: ./
   ```

2. **Test from branch reference**:
   ```yaml
   - uses: your-org/github-project-agent@main
   ```

3. **Test from specific commit or tag**:
   ```yaml
   - uses: your-org/github-project-agent@v1.0.0-beta
   ```

> 📖 **Complete Testing Guide**: See [TESTING_WITHOUT_PUBLISHING.md](TESTING_WITHOUT_PUBLISHING.md) for detailed instructions on testing in GitHub Actions without making your repository public.

### Marketplace Usage (After Publishing)

Once published to the GitHub Actions Marketplace, users can reference it by version:

```yaml
- name: Run GitHub Project Agent
  uses: your-org/github-project-agent@v1.0.0
  with:
    mode: 'mcp'
    agent: 'Task Validator'
    issue: '123'
    github_token: ${{ secrets.GITHUB_TOKEN }}
    litellm_base_url: ${{ secrets.LITELLM_BASE_URL }}
    llm_model: ${{ secrets.LLM_MODEL }}
```

### Required Inputs

| Input | Required | Description |
|-------|----------|-------------|
| `mode` | Yes | Agent mode: `validate`, `monitor`, `roast`, `mcp` |
| `litellm_base_url` | Yes | LiteLLM/vLLM server URL |

### Auto-Configured (No Setup Needed!)

These inputs are **automatically detected** from your workflow context:

- `github_token`: Uses `secrets.GITHUB_TOKEN` automatically (provided by GitHub Actions)
- `github_owner`: Auto-detected from `github.repository_owner`
- `github_repo`: Auto-detected from `github.event.repository.name`
- `llm_model`: Defaults to `gpt-4` if not specified

### Optional Inputs

- `agent`: Agent name for MCP mode (e.g., "Task Validator", "Progress Reporter")
- `issue`: Issue number to process
- `github_token`: GitHub token (auto-uses `secrets.GITHUB_TOKEN` if not provided)
- `github_owner`: GitHub organization/username (auto-detected from workflow context)
- `github_repo`: Repository name (auto-detected from workflow context). **Not needed in project mode** - system searches across all repos automatically
- `github_project_id`: GitHub Project ID (for project mode with multiple repositories)
- `github_repos`: Comma-separated repos for project mode (e.g., `"owner/repo1,owner/repo2"`). Required when using `github_project_id`
- `llm_model`: LLM model name (defaults to `gpt-4`)
- `llm_api_key`: LLM API key (if required)
- `guidelines_path`: Path to guidelines file (default: `.github/task-guidelines.md`)
- `prompts_path`: Path to prompts directory (default: `prompts`)
- `plugins_path`: Path to plugins directory (default: `.github/agents`)
- `stale_threshold_days`: Days before task is stale (default: `7`)

### GitHub App Authentication (Advanced)

For better security and higher rate limits, you can use GitHub App authentication:

```yaml
- uses: your-org/github-project-agent@v1.0.0
  with:
    mode: 'mcp'
    agent: 'Task Validator'
    github_app_id: ${{ secrets.GITHUB_APP_ID }}
    github_app_installation_id: ${{ secrets.GITHUB_APP_INSTALLATION_ID }}
    github_app_private_key: ${{ secrets.GITHUB_APP_PRIVATE_KEY }}
    litellm_base_url: ${{ secrets.LITELLM_BASE_URL }}
```

See [GITHUB_APP_SETUP.md](GITHUB_APP_SETUP.md) for detailed setup instructions.

### Example Workflows

**Auto-validate on issue events:**
```yaml
on:
  issues:
    types: [opened, edited]

jobs:
  validate:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4
      - uses: ./
        with:
          mode: 'mcp'
          agent: 'Task Validator'
          issue: ${{ github.event.issue.number }}
          github_token: ${{ secrets.GITHUB_TOKEN }}
          litellm_base_url: ${{ secrets.LITELLM_BASE_URL }}
          llm_model: ${{ secrets.LLM_MODEL }}
```

**Scheduled progress reports (Project Mode):**
```yaml
on:
  schedule:
    - cron: '0 9 * * 1'  # Every Monday at 9 AM

jobs:
  report:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4
      - uses: ./
        with:
          mode: 'mcp'
          agent: 'Progress Reporter'
          github_project_id: ${{ secrets.GITHUB_PROJECT_ID }}
          github_repos: ${{ secrets.GITHUB_REPOS }}  # "owner/repo1,owner/repo2,owner/repo3"
          # github_repo is NOT needed - system searches across all repos automatically!
          litellm_base_url: ${{ secrets.LITELLM_BASE_URL }}
          llm_model: ${{ secrets.LLM_MODEL }}
```

For detailed information about testing and publishing to the Marketplace, see [MARKETPLACE.md](MARKETPLACE.md).

## Customizing Agent Prompts

All agents use markdown-based prompt templates stored in the `prompts/` directory. This allows you to:

- **Customize agent behavior** without code changes
- **Share prompts** with your team via version control
- **A/B test** different prompt variations
- **Add new agents** easily (see `ADDING_AGENTS.md`)

### Example: Customizing the Roaster

Edit `prompts/roaster.md` to change how the product analysis works:

```markdown
# Product Roast Prompt

You are a brutally honest product advisor...

## Project Context
- Total issues: {{.TotalIssues}}
- Open issues: {{.OpenIssues}}
...
```

The system automatically loads these templates and uses them. If a template file is missing, agents fall back to hardcoded prompts.

See `prompts/README.md` for template syntax and `ADDING_AGENTS.md` for creating new agents.

## Task Format Rules & Guidelines

The agent enforces format rules that can be defined in a markdown file. By default, it looks for `.github/task-guidelines.md` in your repository.

### Markdown Guidelines Format

Create a markdown file (e.g., `.github/task-guidelines.md`) with the following structure:

```markdown
# Task Format Guidelines

## Format Rules

### Required Sections
- Description
- Acceptance Criteria

### Minimum Requirements
- Description Length: 100 characters
- Labels Required: Yes
- Label Prefix: priority:

## Instructions
[Your custom instructions here]

## Examples
[Good and bad examples]
```

The agent will automatically parse this file and use it for validation. See `.github/task-guidelines.md` for a complete example.

**Default rules** (if no guidelines file is found):
- **Description**: Minimum 50 characters
- **Required Sections**: "Description", "Acceptance Criteria"
- **Labels**: Must have a priority label (e.g., `priority:high`)

## How It Works

### Validation Agent
- Scans issues for format compliance
- Uses LLM to intelligently fix formatting issues
- Adds comments explaining what was fixed

### Monitoring Agent
- Tracks open issues assigned to team members
- Detects tasks that haven't been updated in X days
- Generates personalized status check messages using LLM

### Roasting Agent
- Analyzes all issues in the repository
- Provides honest feedback about the product and roadmap
- Generates specific, actionable task suggestions

## LiteLLM/VLLM Integration

The agent uses a standard OpenAI-compatible API endpoint. Configure your litellm or vllm server to expose an endpoint at `/v1/chat/completions`.

Example litellm configuration:
```yaml
model_list:
  - model_name: gpt-4
    litellm_params:
      model: ollama/llama2
      api_base: http://localhost:11434
```

Then point the agent to your litellm server:
```bash
export LITELLM_BASE_URL="http://localhost:4000"
export LLM_MODEL="gpt-4"
```

## GitHub Actions (Self-Hosted Runner)

The agent includes ready-to-use GitHub Actions workflows for automation:

### Setup

1. **Configure your self-hosted runner** to have access to:
   - Your litellm/vllm server (network access)
   - Go 1.21+ installed
   - Required environment variables (set as GitHub Secrets)

2. **Add GitHub Secrets:**
   - `LITELLM_BASE_URL`: Your litellm/vllm endpoint
   - `LLM_MODEL`: Model name (optional, defaults to `gpt-4`)
   - `LLM_API_KEY`: API key if required (optional)
   - `STALE_TASK_THRESHOLD_DAYS`: Days before task is stale (optional, defaults to 7)

3. **Workflows included:**
   - `.github/workflows/agent-validate.yml`: Validates tasks on issue open/edit and daily
   - `.github/workflows/agent-monitor.yml`: Checks for stale tasks every 6 hours
   - `.github/workflows/agent-roast.yml`: Generates weekly product analysis

### Workflow Triggers

- **Validation**: Runs automatically when issues are opened/edited, daily at 9 AM UTC, or manually via workflow_dispatch
- **Monitoring**: Runs every 6 hours or manually via workflow_dispatch
- **Roasting**: Runs weekly on Mondays at 10 AM UTC or manually via workflow_dispatch

### Manual Execution

You can manually trigger any workflow from the GitHub Actions tab, and optionally provide an issue number for validation.

## Building

```bash
go build -o github-project-agent main.go
```

## Running as a Service

You can run the monitor daemon as a systemd service or similar. Example systemd unit:

```ini
[Unit]
Description=GitHub Project Agent Monitor
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/github-project-agent
Environment="GITHUB_TOKEN=..."
Environment="GITHUB_OWNER=..."
Environment="GITHUB_REPO=..."
Environment="LITELLM_BASE_URL=http://localhost:4000"
ExecStart=/path/to/github-project-agent -mode=monitor -daemon
Restart=always

[Install]
WantedBy=multi-user.target
```

## Plugin System

The agent supports a plugin architecture that allows you to add custom agents without modifying the codebase. Simply create `.md` files in `.github/agents/custom/` following the standard format.

**Quick Start:**
1. Create a file `.github/agents/custom/my-agent.md`
2. Define your agent following the format in [PLUGINS.md](PLUGINS.md)
3. Restart the application - your agent will be automatically loaded!

**Example:**
```markdown
# Agent: Code Review Enforcer

**Type**: custom
**Purpose**: Ensure pull requests meet review requirements.

## Trigger
- event: pull_request.opened
- schedule: "0 9 * * *"

## Actions
1. Check if PR has at least 2 approvals
2. Verify all CI checks are passing
3. Add label "ready-to-merge" if all checks pass
```

See [PLUGINS.md](PLUGINS.md) for complete documentation.

## Quick Example: Adding a Custom Agent

Want to add a new agent? It's simple! Here's a real example:

1. **Create agent definition** (`.github/agents/custom/task-summarizer.md`):
```markdown
# Agent: Task Summarizer
**Type**: custom
**Purpose**: Generate concise summaries of tasks.

## Trigger
- manual: true

## Actions
1. Generate summary using LLM
2. Add summary as a comment
```

2. **Create prompt template** (`prompts/summarizer.md`):
```markdown
Create a concise summary of: {{.Title}}
{{.Body}}
```

3. **Test it**:
```bash
go run main.go -mode=mcp -agent="Task Summarizer" -issue=13
```

That's it! See [USER_GUIDE_ADDING_AGENT.md](USER_GUIDE_ADDING_AGENT.md) for a complete walkthrough.

## License

MIT

