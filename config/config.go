package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GitHub struct {
		// Token-based authentication (legacy)
		Token string

		// GitHub App authentication (preferred)
		AppID          int64  // GitHub App ID
		InstallationID int64  // Installation ID
		PrivateKeyPath string // Path to private key file (PEM format)
		PrivateKey     []byte // Private key content (alternative to path)

		Owner     string             // Optional: for single-repo mode
		Repo      string             // Optional: for single-repo mode
		ProjectID string             // Optional: GitHub Project number (for multi-repo mode)
		Repos     []RepositoryConfig // Optional: list of repos for project mode
		BaseURL   string             // Optional: for GitHub Enterprise
		Mode      string             // "repo" or "project" - determines which mode to use
	}

	LLM struct {
		LiteLLMBaseURL string // e.g., "http://localhost:4000"
		Model          string // e.g., "gpt-4", "llama-2", etc.
		APIKey         string // Optional: if required by litellm
		Timeout        time.Duration
	}

	Agent struct {
		StaleTaskThresholdDays int           // Days before a task is considered stale
		CheckInterval          time.Duration // How often to check for stale tasks
		TaskFormatRules        TaskFormatRules
		GuidelinesPath         string // Path to markdown guidelines file
		PromptsPath            string // Path to prompts directory
		PluginsPath            string // Path to plugins directory (.github/agents)
	}
}

type RepositoryConfig struct {
	Owner string
	Name  string
}

type TaskFormatRules struct {
	RequiredSections     []string // e.g., ["Description", "Acceptance Criteria", "Priority"]
	MinDescriptionLength int
	RequireLabels        bool
	LabelPrefix          string // e.g., "priority:" for priority labels
}

func Load() (*Config, error) {
	cfg := &Config{}

	// GitHub config
	cfg.GitHub.Token = getEnv("GITHUB_TOKEN", "")
	cfg.GitHub.Owner = getEnv("GITHUB_OWNER", "")
	cfg.GitHub.Repo = getEnv("GITHUB_REPO", "")
	cfg.GitHub.ProjectID = getEnv("GITHUB_PROJECT_ID", "")
	cfg.GitHub.BaseURL = getEnv("GITHUB_BASE_URL", "https://api.github.com")

	// GitHub App authentication (preferred over token)
	cfg.GitHub.AppID = getEnvInt64("GITHUB_APP_ID", 0)
	cfg.GitHub.InstallationID = getEnvInt64("GITHUB_APP_INSTALLATION_ID", 0)
	cfg.GitHub.PrivateKeyPath = getEnv("GITHUB_APP_PRIVATE_KEY_PATH", "")

	// Try to load private key from path if provided
	if cfg.GitHub.PrivateKeyPath != "" {
		keyData, err := os.ReadFile(cfg.GitHub.PrivateKeyPath)
		if err == nil {
			cfg.GitHub.PrivateKey = keyData
		}
	}

	// Alternative: load private key directly from environment variable
	// (useful for GitHub Actions secrets)
	if len(cfg.GitHub.PrivateKey) == 0 {
		privateKeyEnv := getEnv("GITHUB_APP_PRIVATE_KEY", "")
		if privateKeyEnv != "" {
			// Handle base64 encoded keys or raw PEM
			cfg.GitHub.PrivateKey = []byte(privateKeyEnv)
		}
	}

	// Determine mode: if PROJECT_ID is set, use project mode; otherwise use repo mode
	if cfg.GitHub.ProjectID != "" {
		cfg.GitHub.Mode = "project"
		// Parse repositories from GITHUB_REPOS (comma-separated: owner/repo,owner/repo)
		reposStr := getEnv("GITHUB_REPOS", "")
		if reposStr != "" {
			cfg.GitHub.Repos = parseRepos(reposStr)
		}
	} else {
		cfg.GitHub.Mode = "repo"
	}

	// LLM config
	cfg.LLM.LiteLLMBaseURL = getEnv("LITELLM_BASE_URL", "http://localhost:4000")
	cfg.LLM.Model = getEnv("LLM_MODEL", "gpt-4")
	cfg.LLM.APIKey = getEnv("LLM_API_KEY", "")
	cfg.LLM.Timeout = 30 * time.Second

	// Agent config
	cfg.Agent.StaleTaskThresholdDays = getEnvInt("STALE_TASK_THRESHOLD_DAYS", 7)
	cfg.Agent.CheckInterval = time.Duration(getEnvInt("CHECK_INTERVAL_HOURS", 24)) * time.Hour
	cfg.Agent.GuidelinesPath = getEnv("GUIDELINES_PATH", ".github/task-guidelines.md")
	cfg.Agent.PromptsPath = getEnv("PROMPTS_PATH", "prompts")
	cfg.Agent.PluginsPath = getEnv("PLUGINS_PATH", ".github/agents")

	// Note: PROMPTS_PATH can be comma-separated for multiple paths
	// e.g., "prompts,.github/agents/custom/prompts"

	// Task format rules (defaults, can be overridden by guidelines file)
	cfg.Agent.TaskFormatRules.RequiredSections = []string{"Description", "Acceptance Criteria"}
	cfg.Agent.TaskFormatRules.MinDescriptionLength = 50
	cfg.Agent.TaskFormatRules.RequireLabels = true
	cfg.Agent.TaskFormatRules.LabelPrefix = "priority:"

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

func getEnvInt64(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return result
}

func parseRepos(reposStr string) []RepositoryConfig {
	var repos []RepositoryConfig
	parts := strings.Split(reposStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Format: owner/repo
		repoParts := strings.Split(part, "/")
		if len(repoParts) == 2 {
			repos = append(repos, RepositoryConfig{
				Owner: strings.TrimSpace(repoParts[0]),
				Name:  strings.TrimSpace(repoParts[1]),
			})
		}
	}
	return repos
}
