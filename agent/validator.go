package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/kaskol10/github-project-agent/github"
	"github.com/kaskol10/github-project-agent/guidelines"
	"github.com/kaskol10/github-project-agent/llm"
	"github.com/kaskol10/github-project-agent/prompts"
)

type Validator struct {
	githubClient github.UnifiedClient
	llmClient    *llm.Client
	rules        TaskFormatRules
	guidelines   *guidelines.Guidelines
	promptLoader *prompts.Loader
}

// TaskFormatRules defines the rules for task format validation
type TaskFormatRules struct {
	RequiredSections     []string
	MinDescriptionLength int
	RequireLabels        bool
	LabelPrefix          string
}

func NewValidator(ghClient github.UnifiedClient, llmClient *llm.Client, rules TaskFormatRules, guidelines *guidelines.Guidelines) *Validator {
	// Try to load prompts from prompts/ directory
	promptPath := getPromptPath("prompts")
	promptLoader, _ := prompts.NewLoader(promptPath) // Ignore error, will use fallback

	v := &Validator{
		githubClient: ghClient,
		llmClient:    llmClient,
		rules:        rules,
		guidelines:   guidelines,
		promptLoader: promptLoader,
	}

	// Override rules with guidelines if available
	if guidelines != nil {
		v.rules.RequiredSections = guidelines.FormatRules.RequiredSections
		if len(v.rules.RequiredSections) == 0 {
			v.rules.RequiredSections = rules.RequiredSections // Fallback to defaults
		}
		if guidelines.FormatRules.MinDescriptionLength > 0 {
			v.rules.MinDescriptionLength = guidelines.FormatRules.MinDescriptionLength
		}
		v.rules.RequireLabels = guidelines.FormatRules.RequireLabels || rules.RequireLabels
		if guidelines.FormatRules.LabelPrefix != "" {
			v.rules.LabelPrefix = guidelines.FormatRules.LabelPrefix
		}
	}

	return v
}

func (v *Validator) ValidateAndFix(ctx context.Context, issue *github.Issue) (bool, string, error) {
	violations := v.checkFormat(issue)

	if len(violations) == 0 {
		return true, "", nil
	}

	// Use LLM to fix the issue
	fixedBody, err := v.fixWithLLM(ctx, issue, violations)
	if err != nil {
		return false, "", fmt.Errorf("failed to fix with LLM: %w", err)
	}

	// Preserve original content and add agent modification notice
	updatedBody := v.preserveOriginalWithModifications(issue.Body, fixedBody, violations)

	// Extract owner and repo from issue URL if in project mode
	owner, repo := extractRepoFromURL(issue.URL)

	// Update the issue
	if err := v.githubClient.UpdateIssue(ctx, owner, repo, issue.Number, nil, &updatedBody); err != nil {
		return false, "", fmt.Errorf("failed to update issue: %w", err)
	}

	comment := fmt.Sprintf("🤖 **Agent**: I've updated this task to follow our format guidelines.\n\nIssues fixed:\n%s",
		strings.Join(violations, "\n- "))

	if err := v.githubClient.AddComment(ctx, owner, repo, issue.Number, comment); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to add comment: %v\n", err)
	}

	return false, comment, nil
}

func (v *Validator) checkFormat(issue *github.Issue) []string {
	var violations []string

	// Check description length
	if len(issue.Body) < v.rules.MinDescriptionLength {
		violations = append(violations, fmt.Sprintf("Description too short (minimum %d characters)", v.rules.MinDescriptionLength))
	}

	// Check required sections
	bodyLower := strings.ToLower(issue.Body)
	for _, section := range v.rules.RequiredSections {
		if !strings.Contains(bodyLower, strings.ToLower(section)) {
			violations = append(violations, fmt.Sprintf("Missing required section: %s", section))
		}
	}

	// Check labels if required
	if v.rules.RequireLabels {
		hasPriorityLabel := false
		for _, label := range issue.Labels {
			if strings.HasPrefix(label, v.rules.LabelPrefix) {
				hasPriorityLabel = true
				break
			}
		}
		if !hasPriorityLabel {
			violations = append(violations, fmt.Sprintf("Missing priority label (should start with '%s')", v.rules.LabelPrefix))
		}
	}

	return violations
}

func (v *Validator) fixWithLLM(ctx context.Context, issue *github.Issue, violations []string) (string, error) {
	// Try to use template, fallback to hardcoded prompt
	var prompt string
	if v.promptLoader != nil && v.promptLoader.HasTemplate("validator") {
		guidelinesText := ""
		instructionsText := ""
		if v.guidelines != nil {
			guidelinesText = v.guidelines.RawContent
			instructionsText = v.guidelines.Instructions
		}

		data := map[string]interface{}{
			"Title":                issue.Title,
			"Body":                 issue.Body,
			"Violations":           violations,
			"MinDescriptionLength": v.rules.MinDescriptionLength,
			"RequiredSections":     strings.Join(v.rules.RequiredSections, ", "),
			"LabelPrefix":          v.rules.LabelPrefix,
			"Guidelines":           guidelinesText,
			"Instructions":         instructionsText,
		}

		rendered, err := v.promptLoader.Render("validator", data)
		if err == nil {
			prompt = rendered
		}
	}

	// Fallback to hardcoded prompt if template not available
	if prompt == "" {
		guidelinesText := ""
		if v.guidelines != nil {
			guidelinesText = fmt.Sprintf("\n\nProject Guidelines:\n%s", v.guidelines.RawContent)
			if v.guidelines.Instructions != "" {
				guidelinesText = fmt.Sprintf("\n\nInstructions:\n%s", v.guidelines.Instructions)
			}
		}

		prompt = fmt.Sprintf(`You are a task format enforcer for a GitHub project. Fix the following task to comply with the format guidelines.%s

Current task:
Title: %s
Body: %s

Format violations:
%s

Required format:
- Description: At least %d characters
- Required sections: %s
- Priority label: Must have a label starting with "%s"

Please rewrite the task body to fix all violations while preserving the original intent and information. Return ONLY the fixed body text, no explanations.`,
			guidelinesText,
			issue.Title,
			issue.Body,
			strings.Join(violations, "\n"),
			v.rules.MinDescriptionLength,
			strings.Join(v.rules.RequiredSections, ", "),
			v.rules.LabelPrefix,
		)
	}

	fixedBody, err := v.llmClient.Prompt(prompt)
	if err != nil {
		return "", err
	}

	// Clean up the response (remove markdown code blocks if present)
	fixedBody = strings.TrimSpace(fixedBody)
	if strings.HasPrefix(fixedBody, "```") {
		lines := strings.Split(fixedBody, "\n")
		if len(lines) > 2 {
			fixedBody = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	return fixedBody, nil
}

// preserveOriginalWithModifications preserves the original issue body and adds
// a clear indication of what was modified by the agent
func (v *Validator) preserveOriginalWithModifications(originalBody, fixedBody string, violations []string) string {
	// Check if the body already has an agent modification notice
	agentNoticeStart := "<!-- 🤖 Agent Modified -->"
	agentNoticeEnd := "<!-- /Agent Modified -->"

	// Remove any existing agent notice from original body
	cleanedOriginal := v.removeExistingAgentNotice(originalBody, agentNoticeStart, agentNoticeEnd)

	// Create the modification notice
	violationsList := ""
	for _, violation := range violations {
		violationsList += fmt.Sprintf("- %s\n", violation)
	}

	// Format: Agent notice at top (collapsible), then fixed content, then original preserved
	modificationNotice := fmt.Sprintf(`%s
<details>
<summary>🤖 <strong>Automatically modified by Agent</strong> - Click to see what changed</summary>

This issue was automatically updated to comply with format guidelines.

**Issues fixed:**
%s
</details>
%s

---

%s

---

<details>
<summary>📋 Original content (preserved for reference)</summary>

%s

</details>
`, agentNoticeStart, violationsList, agentNoticeEnd, fixedBody, cleanedOriginal)

	return modificationNotice
}

// removeExistingAgentNotice removes any existing agent modification notice
func (v *Validator) removeExistingAgentNotice(body, startMarker, endMarker string) string {
	startIdx := strings.Index(body, startMarker)
	if startIdx == -1 {
		return body // No existing notice
	}

	endIdx := strings.Index(body[startIdx:], endMarker)
	if endIdx == -1 {
		return body // Malformed notice, keep as is
	}

	endIdx += startIdx + len(endMarker)

	// Remove the notice and any trailing newlines
	before := strings.TrimRight(body[:startIdx], "\n")
	after := strings.TrimLeft(body[endIdx:], "\n")

	if before == "" {
		return after
	}
	if after == "" {
		return before
	}

	return before + "\n\n" + after
}
