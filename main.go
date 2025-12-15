package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kaskol10/github-project-agent/agent"
	"github.com/kaskol10/github-project-agent/config"
	"github.com/kaskol10/github-project-agent/github"
	"github.com/kaskol10/github-project-agent/guidelines"
	"github.com/kaskol10/github-project-agent/llm"
	"github.com/kaskol10/github-project-agent/mcp"
	"github.com/kaskol10/github-project-agent/plugins"
)

func main() {
	var (
		mode         = flag.String("mode", "validate", "Mode: validate, monitor, roast, all, or mcp")
		issueNumber  = flag.Int("issue", 0, "Issue number to validate (for validate mode)")
		runOnce      = flag.Bool("once", false, "Run once and exit (for monitor mode)")
		daemon       = flag.Bool("daemon", false, "Run as daemon (for monitor mode)")
		agentName    = flag.String("agent", "", "Agent name to execute (for mcp mode)")
		workflowName = flag.String("workflow", "", "Workflow name to execute (for mcp mode)")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate required config
	// Either token or GitHub App credentials must be provided
	var appAuth *github.AppAuth
	if cfg.GitHub.AppID > 0 && cfg.GitHub.InstallationID > 0 && len(cfg.GitHub.PrivateKey) > 0 {
		// Use GitHub App authentication
		var err error
		appAuth, err = github.NewAppAuth(
			cfg.GitHub.AppID,
			cfg.GitHub.InstallationID,
			cfg.GitHub.PrivateKey,
			cfg.GitHub.BaseURL,
		)
		if err != nil {
			log.Fatalf("Failed to create GitHub App authenticator: %v", err)
		}
		log.Println("Using GitHub App authentication")
	} else if cfg.GitHub.Token == "" {
		log.Fatal("Either GITHUB_TOKEN or GitHub App credentials (GITHUB_APP_ID, GITHUB_APP_INSTALLATION_ID, GITHUB_APP_PRIVATE_KEY) must be provided")
	} else {
		log.Println("Using token-based authentication")
	}

	if cfg.GitHub.Owner == "" {
		log.Fatal("GITHUB_OWNER environment variable is required")
	}

	// Validate mode-specific requirements
	if cfg.GitHub.Mode == "project" {
		if cfg.GitHub.ProjectID == "" {
			log.Fatal("GITHUB_PROJECT_ID is required when using project mode")
		}
		if len(cfg.GitHub.Repos) == 0 {
			log.Fatal("GITHUB_REPOS is required when using project mode (format: owner/repo,owner/repo)")
		}
		log.Printf("Using project mode with project ID: %s", cfg.GitHub.ProjectID)
		log.Printf("Monitoring %d repositories", len(cfg.GitHub.Repos))
	} else {
		if cfg.GitHub.Repo == "" {
			log.Fatal("GITHUB_REPO environment variable is required for repo mode")
		}
		log.Printf("Using repo mode: %s/%s", cfg.GitHub.Owner, cfg.GitHub.Repo)
	}

	// Convert RepositoryConfig to Repository for unified client
	repos := make([]github.Repository, len(cfg.GitHub.Repos))
	for i, r := range cfg.GitHub.Repos {
		repos[i] = github.Repository{Owner: r.Owner, Name: r.Name}
	}

	// Initialize unified client (works with both repo and project modes)
	ghClient, err := github.NewUnifiedClientWithAuth(
		cfg.GitHub.Token,
		appAuth,
		cfg.GitHub.Owner,
		cfg.GitHub.Repo,
		cfg.GitHub.ProjectID,
		repos,
		cfg.GitHub.BaseURL,
	)
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}

	llmClient := llm.NewClient(
		cfg.LLM.LiteLLMBaseURL,
		cfg.LLM.Model,
		cfg.LLM.APIKey,
		cfg.LLM.Timeout,
	)

	// Load guidelines if path is specified
	var gd *guidelines.Guidelines
	if cfg.Agent.GuidelinesPath != "" {
		if g, err := guidelines.LoadFromFile(cfg.Agent.GuidelinesPath); err == nil {
			gd = g
			log.Printf("Loaded guidelines from: %s", cfg.Agent.GuidelinesPath)
		} else {
			log.Printf("Warning: Could not load guidelines from %s: %v (using defaults)", cfg.Agent.GuidelinesPath, err)
		}
	}

	// Load plugin agents from .github/agents/ directory
	var pluginAgents []*plugins.PluginAgent
	if cfg.Agent.PluginsPath != "" {
		if pa, err := plugins.LoadPlugins(cfg.Agent.PluginsPath); err == nil {
			pluginAgents = pa
			log.Printf("Loaded %d plugin agents from: %s", len(pluginAgents), cfg.Agent.PluginsPath)
			for _, agent := range pluginAgents {
				log.Printf("  - %s (%s)", agent.Name, agent.Type)
			}
		} else {
			log.Printf("Info: Could not load plugins from %s: %v (continuing without plugins)", cfg.Agent.PluginsPath, err)
		}
	}

	ctx := context.Background()

	switch *mode {
	case "validate":
		if err := runValidate(ctx, ghClient, llmClient, cfg, *issueNumber, gd); err != nil {
			log.Fatalf("Validation failed: %v", err)
		}
	case "monitor":
		if *daemon {
			runMonitorDaemon(ctx, ghClient, llmClient, cfg)
		} else if *runOnce {
			if err := runMonitorOnce(ctx, ghClient, llmClient, cfg); err != nil {
				log.Fatalf("Monitoring failed: %v", err)
			}
		} else {
			log.Fatal("Monitor mode requires either -once or -daemon flag")
		}
	case "roast":
		if err := runRoast(ctx, ghClient, llmClient); err != nil {
			log.Fatalf("Roast failed: %v", err)
		}
	case "all":
		if err := runAll(ctx, ghClient, llmClient, cfg, *issueNumber, gd); err != nil {
			log.Fatalf("Failed: %v", err)
		}
	case "mcp":
		if len(pluginAgents) == 0 {
			log.Fatal("No plugin agents found. Create agents in .github/agents/core/ or .github/agents/custom/")
		}
		if err := runMCP(ctx, ghClient, pluginAgents, *agentName, *workflowName, *issueNumber, llmClient, gd, cfg); err != nil {
			log.Fatalf("MCP execution failed: %v", err)
		}
	default:
		log.Fatalf("Unknown mode: %s. Use: validate, monitor, roast, all, or mcp", *mode)
	}
}

func runValidate(ctx context.Context, ghClient github.UnifiedClient, llmClient *llm.Client, cfg *config.Config, issueNumber int, guidelines *guidelines.Guidelines) error {
	validator := agent.NewValidator(ghClient, llmClient, agent.TaskFormatRules{
		RequiredSections:     cfg.Agent.TaskFormatRules.RequiredSections,
		MinDescriptionLength: cfg.Agent.TaskFormatRules.MinDescriptionLength,
		RequireLabels:        cfg.Agent.TaskFormatRules.RequireLabels,
		LabelPrefix:          cfg.Agent.TaskFormatRules.LabelPrefix,
	}, guidelines)

	if issueNumber > 0 {
		// Validate specific issue
		// In project mode, we need owner/repo - for now, search all repos
		// In repo mode, owner/repo can be empty
		var issue *github.Issue
		var err error

		if ghClient.GetMode() == "project" {
			// In project mode, we'd need to know which repo - for now, list all and find it
			// This is a limitation - in production, you'd want to pass repo info
			allIssues, listErr := ghClient.ListIssues(ctx, "all")
			if listErr != nil {
				return fmt.Errorf("failed to list issues: %w", listErr)
			}
			found := false
			for _, i := range allIssues {
				if i.Number == issueNumber {
					issue = i
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("issue #%d not found in project", issueNumber)
			}
		} else {
			// Repo mode - owner/repo not needed
			issue, err = ghClient.GetIssue(ctx, "", "", issueNumber)
			if err != nil {
				return fmt.Errorf("failed to get issue: %w", err)
			}
		}

		valid, comment, err := validator.ValidateAndFix(ctx, issue)
		if err != nil {
			return err
		}

		if valid {
			fmt.Printf("✅ Issue #%d is valid\n", issueNumber)
		} else {
			fmt.Printf("⚠️  Issue #%d was fixed\n", issueNumber)
			fmt.Printf("Comment: %s\n", comment)
		}
	} else {
		// Validate all open issues
		issues, err := ghClient.ListIssues(ctx, "open")
		if err != nil {
			return fmt.Errorf("failed to list issues: %w", err)
		}

		fmt.Printf("Validating %d open issues...\n", len(issues))
		fixed := 0
		for _, issue := range issues {
			valid, _, err := validator.ValidateAndFix(ctx, issue)
			if err != nil {
				fmt.Printf("Error validating issue #%d: %v\n", issue.Number, err)
				continue
			}
			if !valid {
				fixed++
				fmt.Printf("Fixed issue #%d: %s\n", issue.Number, issue.Title)
			}
		}
		fmt.Printf("✅ Validation complete. Fixed %d issues.\n", fixed)
	}

	return nil
}

func runMonitorOnce(ctx context.Context, ghClient github.UnifiedClient, llmClient *llm.Client, cfg *config.Config) error {
	monitor := agent.NewMonitor(ghClient, llmClient, cfg.Agent.StaleTaskThresholdDays)
	fmt.Println("Checking for stale tasks...")
	return monitor.CheckStaleTasks(ctx)
}

func runMonitorDaemon(ctx context.Context, ghClient github.UnifiedClient, llmClient *llm.Client, cfg *config.Config) {
	monitor := agent.NewMonitor(ghClient, llmClient, cfg.Agent.StaleTaskThresholdDays)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(cfg.Agent.CheckInterval)
	defer ticker.Stop()

	fmt.Printf("Starting monitor daemon (checking every %v)...\n", cfg.Agent.CheckInterval)

	// Run immediately
	if err := monitor.CheckStaleTasks(ctx); err != nil {
		log.Printf("Error checking stale tasks: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			fmt.Println("Checking for stale tasks...")
			if err := monitor.CheckStaleTasks(ctx); err != nil {
				log.Printf("Error checking stale tasks: %v", err)
			}
		case <-sigChan:
			fmt.Println("\nShutting down monitor daemon...")
			return
		}
	}
}

func runRoast(ctx context.Context, ghClient github.UnifiedClient, llmClient *llm.Client) error {
	roaster := agent.NewRoaster(ghClient, llmClient)
	fmt.Println("Roasting your product and generating suggestions...")
	return roaster.RoastAndSuggest(ctx)
}

func runAll(ctx context.Context, ghClient github.UnifiedClient, llmClient *llm.Client, cfg *config.Config, issueNumber int, guidelines *guidelines.Guidelines) error {
	fmt.Println("Running all agent tasks...\n")

	// 1. Validate
	fmt.Println("1. Validating tasks...")
	if err := runValidate(ctx, ghClient, llmClient, cfg, issueNumber, guidelines); err != nil {
		log.Printf("Validation error: %v", err)
	}

	// 2. Monitor
	fmt.Println("\n2. Checking for stale tasks...")
	if err := runMonitorOnce(ctx, ghClient, llmClient, cfg); err != nil {
		log.Printf("Monitoring error: %v", err)
	}

	// 3. Roast
	fmt.Println("\n3. Generating product roast and suggestions...")
	if err := runRoast(ctx, ghClient, llmClient); err != nil {
		log.Printf("Roast error: %v", err)
	}

	fmt.Println("\n✅ All tasks completed!")
	return nil
}

func runMCP(ctx context.Context, ghClient github.UnifiedClient, pluginAgents []*plugins.PluginAgent, agentName, workflowName string, issueNumber int, llmClient *llm.Client, guidelines *guidelines.Guidelines, cfg *config.Config) error {
	mcpInterface := mcp.NewMCPInterface(ghClient, pluginAgents, llmClient, guidelines, cfg)

	if workflowName != "" {
		// Execute workflow
		params := map[string]interface{}{
			"issue_number": issueNumber,
		}

		result, err := mcpInterface.ExecuteWorkflow(ctx, workflowName, params)
		if err != nil {
			return fmt.Errorf("failed to execute workflow: %w", err)
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("Workflow '%s' executed successfully:\n%s\n", workflowName, string(resultJSON))
		return nil
	}

	if agentName != "" {
		// Execute agent
		params := map[string]interface{}{
			"issue_number": issueNumber,
		}

		result, err := mcpInterface.ExecuteAgent(ctx, agentName, params)
		if err != nil {
			return fmt.Errorf("failed to execute agent: %w", err)
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("Agent '%s' executed successfully:\n%s\n", agentName, string(resultJSON))
		return nil
	}

	// List available agents and workflows
	fmt.Println("Available Agents:")
	for _, name := range mcpInterface.ListAgents() {
		fmt.Printf("  - %s\n", name)
	}

	fmt.Println("\nAvailable Workflows:")
	for _, name := range mcpInterface.ListWorkflows() {
		fmt.Printf("  - %s\n", name)
	}

	fmt.Println("\nUsage:")
	fmt.Println("  Execute agent: -mode=mcp -agent='Agent Name' -issue=123")
	fmt.Println("  Execute workflow: -mode=mcp -workflow='Workflow Name' -issue=123")

	return nil
}
