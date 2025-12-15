package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kaskol10/github-project-agent/agent"
	"github.com/kaskol10/github-project-agent/github"
	"github.com/kaskol10/github-project-agent/llm"
	"github.com/kaskol10/github-project-agent/prompts"
)

// PluginExecutor executes plugin-based agents
type PluginExecutor struct {
	llmClient    *llm.Client
	githubClient github.UnifiedClient
	promptLoader *prompts.Loader
}

// NewPluginExecutor creates a new plugin executor
func NewPluginExecutor(llmClient *llm.Client, githubClient github.UnifiedClient, promptLoader *prompts.Loader) *PluginExecutor {
	return &PluginExecutor{
		llmClient:    llmClient,
		githubClient: githubClient,
		promptLoader: promptLoader,
	}
}

// Execute runs a plugin agent
func (e *PluginExecutor) Execute(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["agent"] = pluginAgent.Name
	result["type"] = pluginAgent.Type

	// Execute actions based on agent type
	switch {
	case pluginAgent.Name == "Task Validator" || strings.Contains(strings.ToLower(pluginAgent.Name), "validator"):
		return e.executeValidator(ctx, pluginAgent, params)
	case pluginAgent.Name == "Stale Task Monitor" || strings.Contains(strings.ToLower(pluginAgent.Name), "monitor"):
		return e.executeMonitor(ctx, pluginAgent, params)
	case pluginAgent.Name == "Product Roaster" || strings.Contains(strings.ToLower(pluginAgent.Name), "roaster"):
		return e.executeRoaster(ctx, pluginAgent, params)
	case strings.Contains(strings.ToLower(pluginAgent.Name), "code review") || strings.Contains(strings.ToLower(pluginAgent.Name), "review"):
		return e.executeCodeReview(ctx, pluginAgent, params)
	case strings.Contains(strings.ToLower(pluginAgent.Name), "deployment"):
		return e.executeDeployment(ctx, pluginAgent, params)
	case pluginAgent.Name == "Executive Summary Generator" || strings.Contains(strings.ToLower(pluginAgent.Name), "executive summary"):
		return e.executeExecutiveSummary(ctx, pluginAgent, params)
	case pluginAgent.Name == "Progress Reporter" || strings.Contains(strings.ToLower(pluginAgent.Name), "progress reporter"):
		return e.executeProgressReporter(ctx, pluginAgent, params)
	// Priority Calculator, Dependency Tracker, etc. use generic executor
	// The generic executor intelligently parses actions and executes them
	default:
		// Generic plugin execution
		return e.executeGeneric(ctx, pluginAgent, params)
	}
}

// executeValidator executes a task validator plugin
func (e *PluginExecutor) executeValidator(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	// Try both "issue_number" and "issue" for compatibility
	var issueNum int
	var ok bool
	var specificIssue *github.Issue

	if issueNum, ok = params["issue_number"].(int); !ok {
		// Try "issue" as fallback
		if issueNum, ok = params["issue"].(int); !ok {
			// Try float64 (JSON numbers are often parsed as float64)
			if issueNumFloat, okFloat := params["issue_number"].(float64); okFloat {
				issueNum = int(issueNumFloat)
				ok = true
			} else if issueNumFloat, okFloat := params["issue"].(float64); okFloat {
				issueNum = int(issueNumFloat)
				ok = true
			}
		}
	}

	// If a specific issue is provided, check it first
	if ok && issueNum > 0 {
		issue, err := e.githubClient.GetIssue(ctx, "", "", issueNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get issue: %w", err)
		}
		specificIssue = issue

		// Check if issue already has the "agent-validator" label
		hasValidatorLabel := false
		for _, label := range issue.Labels {
			if label == "agent-validator" {
				hasValidatorLabel = true
				break
			}
		}

		if hasValidatorLabel {
			// Issue already validated, but still check all other issues
			// Continue to validate all other issues in the project
		} else {
			// Issue doesn't have label, so we need to validate all issues
			// Continue to validate all issues
		}
	}

	// Load prompt (for future use)
	_, _ = e.loadPrompt(pluginAgent)

	// Use the actual validator agent to perform validation
	// Create validator rules with defaults
	rules := agent.TaskFormatRules{
		RequiredSections:     []string{"Description", "Acceptance Criteria"},
		MinDescriptionLength: 50,
		RequireLabels:        true,
		LabelPrefix:          "priority:",
	}

	// Create validator instance
	validatorInstance := agent.NewValidator(e.githubClient, e.llmClient, rules, nil)

	// Get all open issues in the project
	allIssues, err := e.githubClient.ListIssues(ctx, "open")
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Filter issues that don't have the "agent-validator" label
	issuesToValidate := make([]*github.Issue, 0)
	for _, issue := range allIssues {
		hasValidatorLabel := false
		for _, label := range issue.Labels {
			if label == "agent-validator" {
				hasValidatorLabel = true
				break
			}
		}
		if !hasValidatorLabel {
			issuesToValidate = append(issuesToValidate, issue)
		}
	}

	if len(issuesToValidate) == 0 {
		// All issues already validated
		result := map[string]interface{}{
			"agent":           pluginAgent.Name,
			"status":          "completed",
			"total_issues":    len(allIssues),
			"validated_count": 0,
			"skipped_count":   len(allIssues),
			"message":         "All issues already validated (have 'agent-validator' label)",
		}
		if specificIssue != nil {
			result["issue"] = specificIssue.Number
			result["title"] = specificIssue.Title
		}
		return result, nil
	}

	// Validate all issues that don't have the label
	var validatedCount, fixedCount int
	var errors []string
	validatedIssues := make([]map[string]interface{}, 0)

	for _, issue := range issuesToValidate {
		// Actually run the validation
		valid, comment, err := validatorInstance.ValidateAndFix(ctx, issue)
		if err != nil {
			errors = append(errors, fmt.Sprintf("issue #%d: %v", issue.Number, err))
			continue
		}

		// Add "agent-validator" label to mark this issue as validated
		owner, repo := extractRepoFromURL(issue.URL)
		if err := e.githubClient.AddLabel(ctx, owner, repo, issue.Number, "agent-validator"); err != nil {
			// Log error but don't fail - label addition is not critical
			fmt.Printf("Warning: failed to add 'agent-validator' label to issue #%d: %v\n", issue.Number, err)
		}

		validatedCount++
		if !valid {
			fixedCount++
		}

		validatedIssues = append(validatedIssues, map[string]interface{}{
			"number":    issue.Number,
			"title":     issue.Title,
			"validated": valid,
			"fixed":     !valid,
			"comment":   comment,
		})
	}

	// Build result
	result := map[string]interface{}{
		"agent":            pluginAgent.Name,
		"status":           "completed",
		"total_issues":     len(allIssues),
		"validated_count":  validatedCount,
		"fixed_count":      fixedCount,
		"skipped_count":    len(allIssues) - validatedCount,
		"validated_issues": validatedIssues,
		"message":          fmt.Sprintf("Validated %d issues (%d fixed, %d already valid), %d skipped (already validated)", validatedCount, fixedCount, validatedCount-fixedCount, len(allIssues)-validatedCount),
	}

	if specificIssue != nil {
		result["requested_issue"] = specificIssue.Number
		result["requested_title"] = specificIssue.Title
	}

	if len(errors) > 0 {
		result["errors"] = errors
		result["error_count"] = len(errors)
	}

	return result, nil
}

// executeMonitor executes a stale task monitor plugin
func (e *PluginExecutor) executeMonitor(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	// Get stale threshold from configuration (default: 7 days)
	staleThresholdDays := 7
	if val, ok := pluginAgent.Config["stale_threshold_days"]; ok {
		if days, ok := val.(int); ok {
			staleThresholdDays = days
		} else if daysFloat, ok := val.(float64); ok {
			staleThresholdDays = int(daysFloat)
		}
	}

	threshold := time.Now().AddDate(0, 0, -staleThresholdDays)
	var issuesToCheck []*github.Issue
	var checkedIssue *github.Issue

	// Check if a specific issue was provided
	issueNum, hasIssue := e.extractIssueNumber(params)
	if hasIssue {
		// Monitor specific issue
		issue, err := e.githubClient.GetIssue(ctx, "", "", issueNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get issue: %w", err)
		}
		checkedIssue = issue
		issuesToCheck = []*github.Issue{issue}
	} else {
		// Monitor all open issues
		allIssues, err := e.githubClient.ListIssues(ctx, "open")
		if err != nil {
			return nil, fmt.Errorf("failed to list issues: %w", err)
		}
		issuesToCheck = allIssues
	}

	// Track results
	var staleIssues []int
	var commentedIssues []int
	var errors []string

	// Check each issue
	for _, issue := range issuesToCheck {
		// Only check issues that are assigned
		if issue.Assignee == "" {
			continue
		}

		// Check if stale
		if issue.UpdatedAt.Before(threshold) {
			staleIssues = append(staleIssues, issue.Number)
			daysStale := int(time.Since(issue.UpdatedAt).Hours() / 24)

			// Generate message using LLM
			var prompt string
			templateName := e.extractTemplateName(pluginAgent)

			// Prepare data for prompt template
			data := map[string]interface{}{
				"Title":       issue.Title,
				"Number":      issue.Number,
				"Assignee":    issue.Assignee,
				"LastUpdated": issue.UpdatedAt.Format("2006-01-02"),
				"DaysStale":   daysStale,
				"URL":         issue.URL,
			}

			// Try to load and render prompt template
			if e.promptLoader != nil && templateName != "" && e.promptLoader.HasTemplate(templateName) {
				rendered, err := e.promptLoader.Render(templateName, data)
				if err == nil {
					prompt = rendered
				}
			}

			// Fallback prompt
			if prompt == "" {
				prompt = fmt.Sprintf(`Generate a friendly but professional message to check on the progress of a GitHub task.

Task details:
- Title: %s
- Number: #%d
- Assigned to: %s
- Last updated: %s (%d days ago)
- URL: %s

The task has been in progress for %d days without updates. Ask for a status update in a friendly, non-pushy way. Keep it concise (2-3 sentences). Return ONLY the message text.`,
					issue.Title,
					issue.Number,
					issue.Assignee,
					issue.UpdatedAt.Format("2006-01-02"),
					daysStale,
					issue.URL,
					daysStale,
				)
			}

			// Generate message using LLM
			message, err := e.llmClient.Prompt(prompt)
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
			}

			// Format message with agent prefix
			message = fmt.Sprintf("🤖 **%s**: %s", pluginAgent.Name, message)

			// Add comment to issue
			owner, repo := extractRepoFromURL(issue.URL)
			if err := e.githubClient.AddComment(ctx, owner, repo, issue.Number, message); err != nil {
				errors = append(errors, fmt.Sprintf("issue #%d: %v", issue.Number, err))
			} else {
				commentedIssues = append(commentedIssues, issue.Number)
			}
		}
	}

	// Build result
	result := map[string]interface{}{
		"agent":            pluginAgent.Name,
		"status":           "monitored",
		"total_checked":    len(issuesToCheck),
		"stale_issues":     staleIssues,
		"commented_issues": commentedIssues,
		"stale_threshold":  fmt.Sprintf("%d days", staleThresholdDays),
	}

	if checkedIssue != nil {
		result["issue"] = checkedIssue.Number
		result["title"] = checkedIssue.Title
		if len(staleIssues) > 0 {
			result["is_stale"] = true
			result["days_stale"] = int(time.Since(checkedIssue.UpdatedAt).Hours() / 24)
			result["message"] = fmt.Sprintf("Issue #%d is stale and has been commented", checkedIssue.Number)
		} else {
			result["is_stale"] = false
			result["message"] = fmt.Sprintf("Issue #%d is not stale", checkedIssue.Number)
		}
	} else {
		result["total_issues"] = len(issuesToCheck)
		result["message"] = fmt.Sprintf("Checked %d issues, found %d stale, commented on %d", len(issuesToCheck), len(staleIssues), len(commentedIssues))
	}

	if len(errors) > 0 {
		result["errors"] = errors
		result["warning"] = fmt.Sprintf("Some comments failed: %d errors", len(errors))
	}

	return result, nil
}

// executeRoaster executes a product roaster plugin
func (e *PluginExecutor) executeRoaster(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	// Load prompt (for future use)
	_, _ = e.loadPrompt(pluginAgent)

	// Get all issues
	issues, err := e.githubClient.ListIssues(ctx, "all")
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	result := map[string]interface{}{
		"agent":        pluginAgent.Name,
		"status":       "analyzed",
		"total_issues": len(issues),
		"message":      "Product analysis completed",
	}

	return result, nil
}

// executeCodeReview executes a code review plugin
func (e *PluginExecutor) executeCodeReview(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	// This would implement code review logic
	// For now, return a placeholder
	result := map[string]interface{}{
		"agent":   pluginAgent.Name,
		"status":  "reviewed",
		"message": "Code review plugin executed (custom implementation needed)",
	}

	return result, nil
}

// executeDeployment executes a deployment checker plugin
func (e *PluginExecutor) executeDeployment(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	// This would implement deployment checking logic
	// For now, return a placeholder
	result := map[string]interface{}{
		"agent":   pluginAgent.Name,
		"status":  "checked",
		"message": "Deployment checker plugin executed (custom implementation needed)",
	}

	return result, nil
}

// executeExecutiveSummary generates an executive summary for C-level stakeholders
func (e *PluginExecutor) executeExecutiveSummary(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	// Get all issues for analysis
	issues, err := e.githubClient.ListIssues(ctx, "open")
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Calculate metrics
	totalIssues := len(issues)
	var openIssues, inProgress, completed, blocked int
	issuesByStatus := make(map[string]int)

	for _, issue := range issues {
		issuesByStatus[issue.State]++
		if issue.State == "open" {
			openIssues++
			// Check if blocked (has "blocked" label or similar)
			for _, label := range issue.Labels {
				if strings.Contains(strings.ToLower(label), "blocked") {
					blocked++
					break
				}
			}
		}
	}

	// Get completed issues
	closedIssues, _ := e.githubClient.ListIssues(ctx, "closed")
	completed = len(closedIssues)

	// Prepare data for prompt
	data := map[string]interface{}{
		"TotalIssues":    totalIssues,
		"OpenIssues":     openIssues,
		"InProgress":     inProgress,
		"Completed":      completed,
		"Blocked":        blocked,
		"IssuesByStatus": formatIssuesByStatus(issuesByStatus),
		"RecentIssues":   formatRecentIssues(issues[:min(10, len(issues))]),
		"Date":           time.Now().Format("2006-01-02"),
	}

	// Load and render prompt template
	var prompt string
	templateName := e.extractTemplateName(pluginAgent)

	if e.promptLoader != nil && templateName != "" && e.promptLoader.HasTemplate(templateName) {
		rendered, err := e.promptLoader.Render(templateName, data)
		if err == nil {
			prompt = rendered
		}
	}

	// Fallback prompt
	if prompt == "" {
		prompt = fmt.Sprintf(`Create an executive summary for this project:

Total Issues: %d
Open: %d
Completed: %d
Blocked: %d

Provide a high-level strategic overview focusing on business impact, risks, and opportunities.`,
			totalIssues, openIssues, completed, blocked)
	}

	// Generate summary using LLM
	summary, err := e.llmClient.Prompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate executive summary: %w", err)
	}

	// Clean up response
	summary = cleanMarkdownResponse(summary)

	// Create summary issue
	issueTitle := fmt.Sprintf("Executive Summary - %s", time.Now().Format("2006-01-02"))

	// Get owner/repo from first issue (optional - CreateIssue can handle empty in project mode)
	var owner, repo string
	if len(issues) > 0 {
		owner, repo = extractRepoFromURL(issues[0].URL)
	} else {
		// Try to get any issue to determine repo
		if len(closedIssues) > 0 {
			owner, repo = extractRepoFromURL(closedIssues[0].URL)
		}
	}

	// Always try to create issue (UnifiedClient handles empty owner/repo in project mode)
	labels := []string{"automated", "executive-summary", "report"}
	newIssue, err := e.githubClient.CreateIssue(ctx, owner, repo, issueTitle, summary, labels)
	if err == nil {
		result := map[string]interface{}{
			"agent":                pluginAgent.Name,
			"status":               "completed",
			"summary":              summary,
			"issue_created":        true,
			"created_issue_number": newIssue.Number,
			"created_issue_url":    newIssue.URL,
			"metrics": map[string]interface{}{
				"total_issues": totalIssues,
				"open":         openIssues,
				"completed":    completed,
				"blocked":      blocked,
			},
			"message": fmt.Sprintf("Executive summary generated and issue #%d created", newIssue.Number),
		}
		return result, nil
	}
	// If issue creation fails, still return summary
	fmt.Printf("Warning: failed to create executive summary issue: %v\n", err)

	// Fallback: return summary even if issue creation failed
	result := map[string]interface{}{
		"agent":   pluginAgent.Name,
		"status":  "completed",
		"summary": summary,
		"metrics": map[string]interface{}{
			"total_issues": totalIssues,
			"open":         openIssues,
			"completed":    completed,
			"blocked":      blocked,
		},
		"message": "Executive summary generated successfully (issue creation failed or repo not determined)",
	}

	return result, nil
}

// executePriorityCalculator calculates and suggests task priority
func (e *PluginExecutor) executePriorityCalculator(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	// Get issue number
	issueNum, hasIssue := e.extractIssueNumber(params)
	if !hasIssue {
		return nil, fmt.Errorf("issue number required (use -issue=123)")
	}

	// Get issue
	issue, err := e.githubClient.GetIssue(ctx, "", "", issueNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	// Prepare data for prompt
	data := map[string]interface{}{
		"Title":        issue.Title,
		"Body":         issue.Body,
		"Labels":       strings.Join(issue.Labels, ", "),
		"State":        issue.State,
		"Assignee":     issue.Assignee,
		"CreatedAt":    issue.CreatedAt.Format("2006-01-02"),
		"Dependencies": extractDependenciesFromBody(issue.Body),
	}

	// Load and render prompt template
	var prompt string
	templateName := e.extractTemplateName(pluginAgent)

	if e.promptLoader != nil && templateName != "" && e.promptLoader.HasTemplate(templateName) {
		rendered, err := e.promptLoader.Render(templateName, data)
		if err == nil {
			prompt = rendered
		}
	}

	// Fallback prompt
	if prompt == "" {
		prompt = fmt.Sprintf(`Analyze this task and calculate its priority (P0, P1, P2, P3):

Title: %s
Body: %s
Labels: %s

Consider: business value, effort, dependencies, strategic alignment, urgency.`,
			issue.Title, issue.Body, strings.Join(issue.Labels, ", "))
	}

	// Generate priority assessment
	assessment, err := e.llmClient.Prompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate priority: %w", err)
	}

	// Clean up response
	assessment = cleanMarkdownResponse(assessment)

	// Extract suggested priority from assessment
	suggestedPriority := extractPriorityFromAssessment(assessment)

	// Add comment with assessment
	owner, repo := extractRepoFromURL(issue.URL)
	comment := fmt.Sprintf("🎯 **Priority Assessment** (Generated by %s)\n\n%s", pluginAgent.Name, assessment)
	if err := e.githubClient.AddComment(ctx, owner, repo, issueNum, comment); err != nil {
		// Log error but don't fail - assessment was generated
		fmt.Printf("Warning: failed to add priority comment: %v\n", err)
	}

	// Optionally apply priority label if configured
	if suggestedPriority != "" {
		// This would require label management - for now just return the suggestion
	}

	result := map[string]interface{}{
		"agent":              pluginAgent.Name,
		"issue":              issueNum,
		"title":              issue.Title,
		"status":             "completed",
		"suggested_priority": suggestedPriority,
		"assessment":         assessment,
		"message":            fmt.Sprintf("Priority assessment generated for issue #%d", issueNum),
	}

	return result, nil
}

// executeDependencyTracker analyzes and tracks task dependencies
func (e *PluginExecutor) executeDependencyTracker(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	// Get issue number
	issueNum, hasIssue := e.extractIssueNumber(params)
	if !hasIssue {
		return nil, fmt.Errorf("issue number required (use -issue=123)")
	}

	// Get issue
	issue, err := e.githubClient.GetIssue(ctx, "", "", issueNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	// Extract dependencies from body
	dependencies := extractDependenciesFromBody(issue.Body)
	blockers := extractBlockersFromBody(issue.Body)

	// Prepare data for prompt
	data := map[string]interface{}{
		"Title":        issue.Title,
		"Body":         issue.Body,
		"Number":       issueNum,
		"Labels":       strings.Join(issue.Labels, ", "),
		"State":        issue.State,
		"Dependencies": formatDependencies(dependencies),
		"Blockers":     formatDependencies(blockers),
		"Blocked":      len(dependencies) > 0,
		"Blocking":     len(blockers) > 0,
	}

	// Load and render prompt template
	var prompt string
	templateName := e.extractTemplateName(pluginAgent)

	if e.promptLoader != nil && templateName != "" && e.promptLoader.HasTemplate(templateName) {
		rendered, err := e.promptLoader.Render(templateName, data)
		if err == nil {
			prompt = rendered
		}
	}

	// Fallback prompt
	if prompt == "" {
		prompt = fmt.Sprintf(`Analyze dependencies for this task:

Title: %s
Body: %s

Identify: dependencies (depends on, requires, needs), blockers (blocks, prevents).`,
			issue.Title, issue.Body)
	}

	// Generate dependency analysis
	analysis, err := e.llmClient.Prompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	// Clean up response
	analysis = cleanMarkdownResponse(analysis)

	// Add comment with analysis
	owner, repo := extractRepoFromURL(issue.URL)
	comment := fmt.Sprintf("🔗 **Dependency Analysis** (Generated by %s)\n\n%s", pluginAgent.Name, analysis)
	if err := e.githubClient.AddComment(ctx, owner, repo, issueNum, comment); err != nil {
		fmt.Printf("Warning: failed to add dependency comment: %v\n", err)
	}

	result := map[string]interface{}{
		"agent":        pluginAgent.Name,
		"issue":        issueNum,
		"title":        issue.Title,
		"status":       "completed",
		"dependencies": dependencies,
		"blockers":     blockers,
		"analysis":     analysis,
		"message":      fmt.Sprintf("Dependency analysis completed for issue #%d", issueNum),
	}

	return result, nil
}

// executeProgressReporter generates progress reports for stakeholders
func (e *PluginExecutor) executeProgressReporter(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	// Get all issues
	openIssues, err := e.githubClient.ListIssues(ctx, "open")
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}

	closedIssues, err := e.githubClient.ListIssues(ctx, "closed")
	if err != nil {
		return nil, fmt.Errorf("failed to list closed issues: %w", err)
	}

	allIssues := append(openIssues, closedIssues...)
	totalTasks := len(allIssues)
	completedTasks := len(closedIssues)
	openTasks := len(openIssues)

	// Calculate metrics
	completionRate := 0.0
	if totalTasks > 0 {
		completionRate = float64(completedTasks) / float64(totalTasks) * 100
	}

	// Count blocked tasks
	blockedTasks := 0
	for _, issue := range openIssues {
		for _, label := range issue.Labels {
			if strings.Contains(strings.ToLower(label), "blocked") {
				blockedTasks++
				break
			}
		}
	}

	// Calculate velocity (tasks completed in last 7 days)
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	recentCompleted := 0
	for _, issue := range closedIssues {
		if issue.UpdatedAt.After(sevenDaysAgo) {
			recentCompleted++
		}
	}
	velocity := float64(recentCompleted) / 7.0 // tasks per day

	// Prepare data for prompt
	data := map[string]interface{}{
		"StartDate":       sevenDaysAgo.Format("2006-01-02"),
		"EndDate":         time.Now().Format("2006-01-02"),
		"TotalTasks":      totalTasks,
		"CompletedTasks":  completedTasks,
		"CompletionRate":  fmt.Sprintf("%.1f", completionRate),
		"InProgressTasks": len(openIssues),
		"OpenTasks":       openTasks,
		"BlockedTasks":    blockedTasks,
		"Velocity":        fmt.Sprintf("%.1f", velocity),
		"Trend":           "Stable",                   // Could be calculated from historical data
		"Milestones":      "No milestones configured", // Could be extracted from labels
		"RecentActivity":  formatRecentActivity(closedIssues[:min(5, len(closedIssues))]),
	}

	// Load and render prompt template
	var prompt string
	templateName := e.extractTemplateName(pluginAgent)

	if e.promptLoader != nil && templateName != "" && e.promptLoader.HasTemplate(templateName) {
		rendered, err := e.promptLoader.Render(templateName, data)
		if err == nil {
			prompt = rendered
		}
	}

	// Fallback prompt
	if prompt == "" {
		prompt = fmt.Sprintf(`Create a progress report:

Period: Last 7 days
Total Tasks: %d
Completed: %d (%.1f%%)
Blocked: %d
Velocity: %.1f tasks/day

Provide a comprehensive progress report with metrics, achievements, risks, and recommendations.`,
			totalTasks, completedTasks, completionRate, blockedTasks, velocity)
	}

	// Generate report using LLM
	report, err := e.llmClient.Prompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate progress report: %w", err)
	}

	// Clean up response
	report = cleanMarkdownResponse(report)

	// Create report issue
	issueTitle := fmt.Sprintf("Progress Report - %s", time.Now().Format("2006-01-02"))

	// Get owner/repo from first issue (optional - CreateIssue can handle empty in project mode)
	var owner, repo string
	if len(openIssues) > 0 {
		owner, repo = extractRepoFromURL(openIssues[0].URL)
	} else if len(closedIssues) > 0 {
		owner, repo = extractRepoFromURL(closedIssues[0].URL)
	}

	// Always try to create issue (UnifiedClient handles empty owner/repo in project mode)
	labels := []string{"automated", "progress-report", "report"}
	newIssue, err := e.githubClient.CreateIssue(ctx, owner, repo, issueTitle, report, labels)
	if err == nil {
		result := map[string]interface{}{
			"agent":                pluginAgent.Name,
			"status":               "completed",
			"report":               report,
			"issue_created":        true,
			"created_issue_number": newIssue.Number,
			"created_issue_url":    newIssue.URL,
			"metrics": map[string]interface{}{
				"total_tasks":     totalTasks,
				"completed":       completedTasks,
				"completion_rate": completionRate,
				"blocked":         blockedTasks,
				"velocity":        velocity,
			},
			"message": fmt.Sprintf("Progress report generated and issue #%d created", newIssue.Number),
		}
		return result, nil
	}
	// If issue creation fails, still return report
	fmt.Printf("Warning: failed to create progress report issue: %v\n", err)

	// Fallback: return report even if issue creation failed
	result := map[string]interface{}{
		"agent":  pluginAgent.Name,
		"status": "completed",
		"report": report,
		"metrics": map[string]interface{}{
			"total_tasks":     totalTasks,
			"completed":       completedTasks,
			"completion_rate": completionRate,
			"blocked":         blockedTasks,
			"velocity":        velocity,
		},
		"message": "Progress report generated successfully (issue creation failed or repo not determined)",
	}

	return result, nil
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatIssuesByStatus(statusMap map[string]int) string {
	var parts []string
	for status, count := range statusMap {
		parts = append(parts, fmt.Sprintf("- %s: %d", status, count))
	}
	return strings.Join(parts, "\n")
}

func formatRecentIssues(issues []*github.Issue) string {
	var parts []string
	for i, issue := range issues {
		if i >= 10 {
			break
		}
		parts = append(parts, fmt.Sprintf("- #%d: %s", issue.Number, issue.Title))
	}
	return strings.Join(parts, "\n")
}

func formatRecentActivity(issues []*github.Issue) string {
	var parts []string
	for _, issue := range issues {
		parts = append(parts, fmt.Sprintf("- #%d: %s (Completed: %s)",
			issue.Number, issue.Title, issue.UpdatedAt.Format("2006-01-02")))
	}
	return strings.Join(parts, "\n")
}

func extractDependenciesFromBody(body string) []string {
	// Extract issue numbers mentioned with dependency keywords
	var deps []string
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "depends on") ||
			strings.Contains(lower, "requires") ||
			strings.Contains(lower, "needs") ||
			strings.Contains(lower, "waiting for") {
			// Extract issue numbers (e.g., #123, issue 456)
			// This is a simple extraction - could be enhanced
			if strings.Contains(line, "#") {
				// Extract number after #
				parts := strings.Split(line, "#")
				for i := 1; i < len(parts); i++ {
					numStr := ""
					for _, r := range parts[i] {
						if r >= '0' && r <= '9' {
							numStr += string(r)
						} else {
							break
						}
					}
					if numStr != "" {
						deps = append(deps, numStr)
					}
				}
			}
		}
	}
	return deps
}

func extractBlockersFromBody(body string) []string {
	// Similar to extractDependenciesFromBody but for blockers
	var blockers []string
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "blocks") ||
			strings.Contains(lower, "prevents") {
			if strings.Contains(line, "#") {
				parts := strings.Split(line, "#")
				for i := 1; i < len(parts); i++ {
					numStr := ""
					for _, r := range parts[i] {
						if r >= '0' && r <= '9' {
							numStr += string(r)
						} else {
							break
						}
					}
					if numStr != "" {
						blockers = append(blockers, numStr)
					}
				}
			}
		}
	}
	return blockers
}

func formatDependencies(deps []string) string {
	if len(deps) == 0 {
		return "None identified"
	}
	var parts []string
	for _, dep := range deps {
		parts = append(parts, fmt.Sprintf("- #%s", dep))
	}
	return strings.Join(parts, "\n")
}

func extractPriorityFromAssessment(assessment string) string {
	// Extract priority (P0, P1, P2, P3) from assessment
	assessmentLower := strings.ToLower(assessment)
	if strings.Contains(assessmentLower, "p0") || strings.Contains(assessmentLower, "critical") {
		return "P0"
	}
	if strings.Contains(assessmentLower, "p1") || strings.Contains(assessmentLower, "high") {
		return "P1"
	}
	if strings.Contains(assessmentLower, "p2") || strings.Contains(assessmentLower, "medium") {
		return "P2"
	}
	if strings.Contains(assessmentLower, "p3") || strings.Contains(assessmentLower, "low") {
		return "P3"
	}
	return ""
}

func cleanMarkdownResponse(response string) string {
	// Clean up LLM response
	response = strings.TrimSpace(response)

	// Remove markdown code blocks
	if strings.HasPrefix(response, "```markdown") {
		lines := strings.Split(response, "\n")
		if len(lines) > 2 {
			response = strings.Join(lines[1:len(lines)-1], "\n")
		}
	} else if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) > 2 {
			response = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	// Normalize line breaks
	response = strings.ReplaceAll(response, "\r\n", "\n")
	response = strings.ReplaceAll(response, "\n##", "\n\n##")
	response = strings.ReplaceAll(response, "\n**", "\n\n**")

	// Clean up triple newlines
	for strings.Contains(response, "\n\n\n") {
		response = strings.ReplaceAll(response, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(response)
}

// extractRepoFromURL extracts owner and repo from a GitHub issue URL
func extractRepoFromURL(url string) (string, string) {
	// URL format: https://github.com/owner/repo/issues/123
	parts := strings.Split(url, "/")
	if len(parts) >= 5 {
		return parts[3], parts[4] // owner, repo
	}
	return "", ""
}

// executeGeneric executes a generic plugin using intelligent action parsing
func (e *PluginExecutor) executeGeneric(ctx context.Context, pluginAgent *PluginAgent, params map[string]interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"agent":   pluginAgent.Name,
		"type":    pluginAgent.Type,
		"status":  "executed",
		"actions": pluginAgent.Actions,
	}

	// Try to get issue number if actions suggest we need it
	var issueNum int
	var hasIssue bool
	if issueNum, hasIssue = e.extractIssueNumber(params); hasIssue {
		result["issue"] = issueNum
	}

	// Intelligent action execution based on action descriptions
	// Process actions in order to handle dependencies
	var issue *github.Issue
	var issueErr error

	for _, action := range pluginAgent.Actions {
		actionLower := strings.ToLower(action)

		// Pattern: "Check if task body is long enough" or "Check if task body is longer than threshold"
		if strings.Contains(actionLower, "check") && (strings.Contains(actionLower, "length") || strings.Contains(actionLower, "long") || strings.Contains(actionLower, "threshold")) {
			if hasIssue {
				if issue == nil {
					issue, issueErr = e.githubClient.GetIssue(ctx, "", "", issueNum)
					if issueErr != nil {
						return nil, fmt.Errorf("failed to get issue: %w", issueErr)
					}
				}

				minLength := 200
				if config, ok := pluginAgent.Config["min_length_for_summary"].(int); ok {
					minLength = config
				}

				if len(issue.Body) < minLength {
					result["status"] = "skipped"
					result["message"] = fmt.Sprintf("Task is too short to summarize (%d chars, minimum %d)", len(issue.Body), minLength)
					return result, nil
				}
				result["task_length"] = len(issue.Body)
				result["length_check_passed"] = true
			}
		}

		// Pattern: "Generate summary using LLM" or "Generate concise summary using LLM" or "Use LLM to..."
		if (strings.Contains(actionLower, "llm") || strings.Contains(actionLower, "generate")) &&
			(strings.Contains(actionLower, "summary") || strings.Contains(actionLower, "content") || strings.Contains(actionLower, "text")) {
			if hasIssue {
				if issue == nil {
					issue, issueErr = e.githubClient.GetIssue(ctx, "", "", issueNum)
					if issueErr != nil {
						return nil, fmt.Errorf("failed to get issue: %w", issueErr)
					}
				}

				llmResult := e.executeLLMAction(ctx, pluginAgent, issue, params)
				if llmResult != nil {
					if summary, ok := llmResult["summary"].(string); ok && summary != "" {
						result["summary"] = summary
						result["llm_called"] = true
					} else if content, ok := llmResult["content"].(string); ok && content != "" {
						result["summary"] = content
						result["llm_called"] = true
					}
				}
			}
		}

		// Pattern: "Add comment" or "Add summary as a comment" or "Add X as a comment"
		if strings.Contains(actionLower, "add") && strings.Contains(actionLower, "comment") {
			if hasIssue {
				if issue == nil {
					issue, issueErr = e.githubClient.GetIssue(ctx, "", "", issueNum)
					if issueErr != nil {
						return nil, fmt.Errorf("failed to get issue: %w", issueErr)
					}
				}

				// Get content to add as comment
				var commentContent string
				if summary, ok := result["summary"].(string); ok && summary != "" {
					commentContent = summary
				} else {
					// If no summary, create a basic comment
					commentContent = fmt.Sprintf("🤖 **%s** executed successfully.", pluginAgent.Name)
				}

				if err := e.addCommentToIssue(ctx, pluginAgent, issueNum, commentContent); err == nil {
					result["comment_added"] = true
					if result["message"] == nil {
						result["message"] = fmt.Sprintf("Summary generated and added as comment to issue #%d", issueNum)
					} else {
						// Update message if summary was generated
						if result["summary"] != nil {
							result["message"] = fmt.Sprintf("Summary generated and added as comment to issue #%d", issueNum)
						}
					}
					result["status"] = "completed"
				} else {
					result["comment_error"] = err.Error()
				}
			}
		}
	}

	// If no specific actions matched, return basic execution result
	if result["message"] == nil {
		result["message"] = fmt.Sprintf("Plugin '%s' executed with %d actions", pluginAgent.Name, len(pluginAgent.Actions))
	}

	return result, nil
}

// extractIssueNumber extracts issue number from params (supports multiple formats)
func (e *PluginExecutor) extractIssueNumber(params map[string]interface{}) (int, bool) {
	var issueNum int
	var ok bool

	if issueNum, ok = params["issue_number"].(int); ok && issueNum > 0 {
		return issueNum, true
	}
	if issueNum, ok = params["issue"].(int); ok && issueNum > 0 {
		return issueNum, true
	}
	if issueNumFloat, okFloat := params["issue_number"].(float64); okFloat {
		issueNum = int(issueNumFloat)
		if issueNum > 0 {
			return issueNum, true
		}
	}
	if issueNumFloat, okFloat := params["issue"].(float64); okFloat {
		issueNum = int(issueNumFloat)
		if issueNum > 0 {
			return issueNum, true
		}
	}

	return 0, false
}

// executeLLMAction executes an LLM-based action using the agent's prompt template
func (e *PluginExecutor) executeLLMAction(ctx context.Context, pluginAgent *PluginAgent, issue *github.Issue, params map[string]interface{}) map[string]interface{} {
	// Load and render prompt template
	var prompt string
	templateName := e.extractTemplateName(pluginAgent)

	// Build data map for template rendering
	data := make(map[string]interface{})

	// Add issue data if available
	if issue != nil {
		data["Title"] = issue.Title
		data["Body"] = issue.Body
		data["Labels"] = strings.Join(issue.Labels, ", ")
		data["State"] = issue.State
		data["Assignee"] = issue.Assignee
		if issue.Number > 0 {
			data["Number"] = issue.Number
		}
		if !issue.CreatedAt.IsZero() {
			data["CreatedAt"] = issue.CreatedAt.Format("2006-01-02")
		}
		if !issue.UpdatedAt.IsZero() {
			data["UpdatedAt"] = issue.UpdatedAt.Format("2006-01-02")
		}
	}

	// For agents that need project-wide data (like Executive Summary, Progress Reporter)
	if strings.Contains(strings.ToLower(pluginAgent.Name), "executive") ||
		strings.Contains(strings.ToLower(pluginAgent.Name), "progress") ||
		strings.Contains(strings.ToLower(pluginAgent.Name), "summary") {
		// Gather project statistics
		if stats := e.gatherProjectStats(ctx); stats != nil {
			for k, v := range stats {
				data[k] = v
			}
		}
	}

	// Add any additional params
	for k, v := range params {
		data[k] = v
	}

	// Try to load template - the multi-path loader will search all configured paths
	if e.promptLoader != nil && templateName != "" && e.promptLoader.HasTemplate(templateName) {
		rendered, err := e.promptLoader.Render(templateName, data)
		if err == nil {
			prompt = rendered
		}
	}

	// Fallback to a generic prompt if template not available
	if prompt == "" {
		if issue != nil {
			prompt = fmt.Sprintf(`You are a helpful assistant. Based on the following GitHub issue, provide a concise summary.

Title: %s
Body: %s
Labels: %s
State: %s
Assignee: %s

Provide a clear, concise summary.`,
				issue.Title,
				issue.Body,
				strings.Join(issue.Labels, ", "),
				issue.State,
				issue.Assignee,
			)
		} else {
			// For agents that don't need a specific issue (like Executive Summary)
			// Build a simple data summary for the prompt
			dataSummary := ""
			for k, v := range data {
				dataSummary += fmt.Sprintf("%s: %v\n", k, v)
			}
			prompt = fmt.Sprintf(`You are a helpful assistant. Analyze the following project data and provide insights.

%s

Provide a clear, structured analysis.`, dataSummary)
		}
	}

	// Call LLM
	summary, err := e.llmClient.Prompt(prompt)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("LLM call failed: %v", err),
		}
	}

	// Clean up response
	summary = strings.TrimSpace(summary)

	// Remove markdown code blocks if present
	if strings.HasPrefix(summary, "```markdown") {
		lines := strings.Split(summary, "\n")
		if len(lines) > 2 {
			summary = strings.Join(lines[1:len(lines)-1], "\n")
		}
	} else if strings.HasPrefix(summary, "```") {
		lines := strings.Split(summary, "\n")
		if len(lines) > 2 {
			summary = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	// Normalize line breaks
	summary = strings.ReplaceAll(summary, "\r\n", "\n")
	summary = strings.TrimSpace(summary)

	// Post-process to ensure correct format if LLM didn't follow instructions
	// Check if it starts with "Summary:" or similar instead of "## Task Summary"
	if strings.HasPrefix(summary, "Summary:") || strings.HasPrefix(summary, "summary:") {
		// Try to extract the actual summary content
		parts := strings.SplitN(summary, ":", 2)
		if len(parts) > 1 {
			content := strings.TrimSpace(parts[1])
			// Reformat to proper structure (this is a fallback, prompt should handle it)
			summary = fmt.Sprintf("## Task Summary\n\n**Objective**: %s", content)
		}
	}

	// Ensure it starts with the proper heading
	if !strings.HasPrefix(summary, "## Task Summary") {
		// If it doesn't start with the heading, try to add it
		if strings.Contains(summary, "Objective") || strings.Contains(summary, "objective") {
			// It might have the content but missing the heading
			summary = "## Task Summary\n\n" + summary
		}
	}

	// Ensure proper markdown formatting for GitHub comments
	// GitHub requires double newlines for paragraph breaks
	// Add double newline before headings if not present
	summary = strings.ReplaceAll(summary, "\n##", "\n\n##")
	// Ensure double newline before bold sections
	summary = strings.ReplaceAll(summary, "\n**", "\n\n**")
	// But don't add newline if it's already there
	summary = strings.ReplaceAll(summary, "\n\n\n**", "\n\n**")

	// Clean up any triple+ newlines
	for strings.Contains(summary, "\n\n\n") {
		summary = strings.ReplaceAll(summary, "\n\n\n", "\n\n")
	}

	summary = strings.TrimSpace(summary)

	return map[string]interface{}{
		"summary": summary,
		"content": summary, // Also provide as "content" for flexibility
	}
}

// addCommentToIssue adds a comment to a GitHub issue
func (e *PluginExecutor) addCommentToIssue(ctx context.Context, pluginAgent *PluginAgent, issueNum int, content string) error {
	// Get issue to extract owner/repo
	issue, err := e.githubClient.GetIssue(ctx, "", "", issueNum)
	if err != nil {
		return err
	}

	owner, repo := extractRepoFromURL(issue.URL)

	// Format comment - ensure proper markdown spacing
	// GitHub requires double newlines for proper rendering
	commentPrefix := fmt.Sprintf("🤖 **%s**\n\n", pluginAgent.Name)

	// Ensure content starts with proper spacing
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "\n") {
		content = "\n" + content
	}

	comment := commentPrefix + content

	return e.githubClient.AddComment(ctx, owner, repo, issueNum, comment)
}

// gatherProjectStats gathers project-wide statistics for agents that need them
func (e *PluginExecutor) gatherProjectStats(ctx context.Context) map[string]interface{} {
	stats := make(map[string]interface{})

	// List all open issues
	openIssues, err := e.githubClient.ListIssues(ctx, "open")
	if err == nil {
		stats["TotalOpenTasks"] = len(openIssues)

		// Count by state/status
		inProgress := 0
		blocked := 0
		for _, issue := range openIssues {
			// Check labels for status
			for _, label := range issue.Labels {
				labelLower := strings.ToLower(label)
				if strings.Contains(labelLower, "in progress") || strings.Contains(labelLower, "in-progress") {
					inProgress++
				}
				if strings.Contains(labelLower, "blocked") || strings.Contains(labelLower, "blocker") {
					blocked++
				}
			}
		}
		stats["InProgressTasks"] = inProgress
		stats["BlockedTasks"] = blocked
	}

	// List closed issues for completion metrics
	closedIssues, err := e.githubClient.ListIssues(ctx, "closed")
	if err == nil {
		stats["CompletedTasks"] = len(closedIssues)

		// Calculate completion rate
		total := stats["TotalOpenTasks"].(int) + len(closedIssues)
		if total > 0 {
			completionRate := float64(len(closedIssues)) / float64(total) * 100
			stats["CompletionRate"] = fmt.Sprintf("%.1f", completionRate)
		}
	}

	// Calculate risk count (issues with "risk" or "blocker" labels)
	riskCount := 0
	if openIssues != nil {
		for _, issue := range openIssues {
			for _, label := range issue.Labels {
				labelLower := strings.ToLower(label)
				if strings.Contains(labelLower, "risk") || strings.Contains(labelLower, "blocker") || strings.Contains(labelLower, "critical") {
					riskCount++
					break
				}
			}
		}
	}
	stats["RiskCount"] = riskCount

	return stats
}

// extractTemplateName extracts template name from prompt path
// Supports multiple path formats:
// - "prompts/summarizer.md" -> "summarizer"
// - ".github/agents/custom/prompts/summarizer.md" -> "summarizer"
// - "summarizer.md" -> "summarizer"
// - Relative paths from agent file location
func (e *PluginExecutor) extractTemplateName(pluginAgent *PluginAgent) string {
	if pluginAgent.PromptPath == "" {
		// Try to infer from agent name (e.g., "Task Summarizer" -> "summarizer")
		name := strings.ToLower(pluginAgent.Name)
		name = strings.ReplaceAll(name, " ", "-")
		name = strings.ReplaceAll(name, "task-", "")
		name = strings.ReplaceAll(name, "executive-", "")
		name = strings.ReplaceAll(name, "priority-", "")
		name = strings.ReplaceAll(name, "progress-", "")
		return name
	}

	// Extract template name from path
	// Handle various path formats
	path := pluginAgent.PromptPath

	// Remove .md extension
	path = strings.TrimSuffix(path, ".md")

	// Remove common prefixes
	path = strings.TrimPrefix(path, "prompts/")
	path = strings.TrimPrefix(path, ".github/agents/custom/prompts/")
	path = strings.TrimPrefix(path, ".github/agents/core/prompts/")

	// Get just the filename
	if strings.Contains(path, "/") {
		parts := strings.Split(path, "/")
		path = parts[len(parts)-1]
	}

	return path
}

// loadPrompt loads the prompt for an agent
func (e *PluginExecutor) loadPrompt(pluginAgent *PluginAgent) (string, error) {
	// Try to load from PromptPath if specified
	if pluginAgent.PromptPath != "" && e.promptLoader != nil {
		// Extract template name from path (e.g., "prompts/summarizer.md" -> "summarizer")
		templateName := strings.TrimSuffix(strings.TrimPrefix(pluginAgent.PromptPath, "prompts/"), ".md")
		if e.promptLoader.HasTemplate(templateName) {
			// Return empty string - the template will be rendered with data later
			return "", nil // Signal that template exists
		}
	}

	// Fallback to raw content or return empty
	return "", nil
}
