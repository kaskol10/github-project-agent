package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kaskol10/github-project-agent/github"
	"github.com/kaskol10/github-project-agent/llm"
	"github.com/kaskol10/github-project-agent/prompts"
)

type Monitor struct {
	githubClient      github.UnifiedClient
	llmClient         *llm.Client
	staleThresholdDays int
	promptLoader      *prompts.Loader
}

func NewMonitor(ghClient github.UnifiedClient, llmClient *llm.Client, staleThresholdDays int) *Monitor {
	// Try to load prompts from prompts/ directory
	promptPath := getPromptPath("prompts")
	promptLoader, _ := prompts.NewLoader(promptPath) // Ignore error, will use fallback
	
	return &Monitor{
		githubClient:       ghClient,
		llmClient:          llmClient,
		staleThresholdDays: staleThresholdDays,
		promptLoader:        promptLoader,
	}
}

func (m *Monitor) CheckStaleTasks(ctx context.Context) error {
	issues, err := m.githubClient.ListIssues(ctx, "open")
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}
	
	threshold := time.Now().AddDate(0, 0, -m.staleThresholdDays)
	
	for _, issue := range issues {
		// Only check issues that are assigned and haven't been updated recently
		if issue.Assignee == "" {
			continue
		}
		
		if issue.UpdatedAt.Before(threshold) {
			if err := m.handleStaleTask(ctx, issue); err != nil {
				fmt.Printf("Error handling stale task #%d: %v\n", issue.Number, err)
				continue
			}
		}
	}
	
	return nil
}

func (m *Monitor) handleStaleTask(ctx context.Context, issue *github.Issue) error {
	daysStale := int(time.Since(issue.UpdatedAt).Hours() / 24)
	
	// Try to use template, fallback to hardcoded prompt
	var prompt string
	if m.promptLoader != nil && m.promptLoader.HasTemplate("monitor") {
		data := map[string]interface{}{
			"Title":      issue.Title,
			"Number":     issue.Number,
			"Assignee":   issue.Assignee,
			"LastUpdated": issue.UpdatedAt.Format("2006-01-02"),
			"DaysStale":  daysStale,
			"URL":        issue.URL,
		}
		
		rendered, err := m.promptLoader.Render("monitor", data)
		if err == nil {
			prompt = rendered
		}
	}
	
	// Fallback to hardcoded prompt if template not available
	if prompt == "" {
		prompt = fmt.Sprintf(`Generate a friendly but professional message to check on the progress of a GitHub task. 

Task details:
- Title: %s
- Number: #%d
- Assigned to: %s
- Last updated: %s (%.0f days ago)
- URL: %s

The task has been in progress for %d days without updates. Ask for a status update in a friendly, non-pushy way. Keep it concise (2-3 sentences). Return ONLY the message text.`,
			issue.Title,
			issue.Number,
			issue.Assignee,
			issue.UpdatedAt.Format("2006-01-02"),
			time.Since(issue.UpdatedAt).Hours()/24,
			issue.URL,
			daysStale,
		)
	}
	
	message, err := m.llmClient.Prompt(prompt)
	if err != nil {
		// Fallback to a simple message
		message = fmt.Sprintf("👋 Hey @%s! This task has been in progress for %d days. Could you share a quick status update? Thanks! 🙏", 
			issue.Assignee, daysStale)
	} else {
		// Clean up LLM response
		message = strings.TrimSpace(message)
		if strings.HasPrefix(message, "```") {
			lines := strings.Split(message, "\n")
			if len(lines) > 2 {
				message = strings.Join(lines[1:len(lines)-1], "\n")
			}
		}
		message = fmt.Sprintf("🤖 **Agent**: %s", message)
	}
	
	owner, repo := extractRepoFromURL(issue.URL)
	return m.githubClient.AddComment(ctx, owner, repo, issue.Number, message)
}

// extractRepoFromURL extracts owner and repo from GitHub issue URL
func extractRepoFromURL(url string) (owner, repo string) {
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		for i, part := range parts {
			if part == "github.com" && i+2 < len(parts) {
				return parts[i+1], parts[i+2]
			}
		}
	}
	return "", ""
}

