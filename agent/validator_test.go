package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/kaskol10/github-project-agent/github"
)

// mockLLMClient is a mock implementation of the LLM client for testing
// We'll need to create a wrapper that matches the actual llm.Client structure
// For now, we'll test without the LLM integration or use a different approach

// mockGitHubClient is a mock implementation of the GitHub client for testing
type mockGitHubClient struct {
	updatedIssues map[int]*github.Issue
	comments      map[int][]string
}

func newMockGitHubClient() *mockGitHubClient {
	return &mockGitHubClient{
		updatedIssues: make(map[int]*github.Issue),
		comments:      make(map[int][]string),
	}
}

func (m *mockGitHubClient) ListIssues(ctx context.Context, state string) ([]*github.Issue, error) {
	return nil, nil
}

func (m *mockGitHubClient) GetIssue(ctx context.Context, owner, repo string, number int) (*github.Issue, error) {
	return nil, nil
}

func (m *mockGitHubClient) UpdateIssue(ctx context.Context, owner, repo string, number int, title, body *string) error {
	if m.updatedIssues[number] == nil {
		m.updatedIssues[number] = &github.Issue{Number: number}
	}
	if body != nil {
		m.updatedIssues[number].Body = *body
	}
	if title != nil {
		m.updatedIssues[number].Title = *title
	}
	return nil
}

func (m *mockGitHubClient) AddComment(ctx context.Context, owner, repo string, number int, comment string) error {
	if m.comments[number] == nil {
		m.comments[number] = []string{}
	}
	m.comments[number] = append(m.comments[number], comment)
	return nil
}

func (m *mockGitHubClient) CreateIssue(ctx context.Context, owner, repo, title, body string, labels []string) (*github.Issue, error) {
	return nil, nil
}

func (m *mockGitHubClient) GetMode() string {
	return "repo"
}

func TestValidator_CheckFormat(t *testing.T) {
	tests := []struct {
		name       string
		issue      *github.Issue
		rules      TaskFormatRules
		wantErrors []string
	}{
		{
			name: "valid issue with all requirements",
			issue: &github.Issue{
				Title:  "Service Mesh on K8s Cluster",
				Body:   "Deploy a Service Mesh on K8s clusters with the aim to improve service-to-service communication reliability and observability.\n\n## Description\n\nThis task involves deploying a service mesh solution.\n\n## Acceptance Criteria\n\n- Service mesh deployed\n- Monitoring enabled",
				Labels: []string{"priority:high"},
			},
			rules: TaskFormatRules{
				RequiredSections:     []string{"Description", "Acceptance Criteria"},
				MinDescriptionLength: 50,
				RequireLabels:        true,
				LabelPrefix:          "priority:",
			},
			wantErrors: []string{},
		},
		{
			name: "issue with short description",
			issue: &github.Issue{
				Title:  "Service Mesh on K8s Cluster",
				Body:   "Deploy a Service Mesh.",
				Labels: []string{"priority:high"},
			},
			rules: TaskFormatRules{
				RequiredSections:     []string{"Description", "Acceptance Criteria"},
				MinDescriptionLength: 50,
				RequireLabels:        true,
				LabelPrefix:          "priority:",
			},
			wantErrors: []string{
				"Description too short (minimum 50 characters)",
				"Missing required section: Description",
				"Missing required section: Acceptance Criteria",
			},
		},
		{
			name: "issue missing required sections",
			issue: &github.Issue{
				Title:  "Service Mesh on K8s Cluster",
				Body:   "Deploy a Service Mesh on K8s clusters with the aim to improve service-to-service communication reliability and observability.",
				Labels: []string{"priority:high"},
			},
			rules: TaskFormatRules{
				RequiredSections:     []string{"Description", "Acceptance Criteria"},
				MinDescriptionLength: 50,
				RequireLabels:        true,
				LabelPrefix:          "priority:",
			},
			wantErrors: []string{
				"Missing required section: Description",
				"Missing required section: Acceptance Criteria",
			},
		},
		{
			name: "issue missing priority label",
			issue: &github.Issue{
				Title:  "Service Mesh on K8s Cluster",
				Body:   "Deploy a Service Mesh on K8s clusters with the aim to improve service-to-service communication reliability and observability.\n\n## Description\n\nThis task involves deploying a service mesh solution.\n\n## Acceptance Criteria\n\n- Service mesh deployed",
				Labels: []string{"bug"},
			},
			rules: TaskFormatRules{
				RequiredSections:     []string{"Description", "Acceptance Criteria"},
				MinDescriptionLength: 50,
				RequireLabels:        true,
				LabelPrefix:          "priority:",
			},
			wantErrors: []string{
				"Missing priority label (should start with 'priority:')",
			},
		},
		{
			name: "real-world example - Service Mesh task",
			issue: &github.Issue{
				Title:  "Service Mesh on K8s Cluster",
				Body:   "Deploy a Service Mesh on K8s clusters with the aim to improve service-to-service communication reliability and observability.",
				Labels: []string{},
			},
			rules: TaskFormatRules{
				RequiredSections:     []string{"Description", "Acceptance Criteria"},
				MinDescriptionLength: 50,
				RequireLabels:        true,
				LabelPrefix:          "priority:",
			},
			wantErrors: []string{
				"Missing required section: Description",
				"Missing required section: Acceptance Criteria",
				"Missing priority label (should start with 'priority:')",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{
				rules: tt.rules,
			}

			gotErrors := v.checkFormat(tt.issue)

			if len(gotErrors) != len(tt.wantErrors) {
				t.Errorf("checkFormat() returned %d errors, want %d", len(gotErrors), len(tt.wantErrors))
				t.Errorf("Got errors: %v", gotErrors)
				t.Errorf("Want errors: %v", tt.wantErrors)
				return
			}

			// Check that all expected errors are present
			errorMap := make(map[string]bool)
			for _, err := range gotErrors {
				errorMap[err] = true
			}

			for _, wantErr := range tt.wantErrors {
				if !errorMap[wantErr] {
					t.Errorf("checkFormat() missing expected error: %s", wantErr)
				}
			}
		})
	}
}

func TestValidator_PreserveOriginalWithModifications(t *testing.T) {
	tests := []struct {
		name          string
		originalBody  string
		fixedBody     string
		violations    []string
		wantContains  []string
		wantNotContains []string
	}{
		{
			name:         "preserves original content",
			originalBody: "Deploy a Service Mesh on K8s clusters with the aim to improve service-to-service communication reliability and observability.",
			fixedBody:    "## Description\n\nDeploy a Service Mesh on K8s clusters with the aim to improve service-to-service communication reliability and observability.\n\n## Acceptance Criteria\n\n- Service mesh deployed\n- Monitoring enabled",
			violations: []string{
				"Missing required section: Description",
				"Missing required section: Acceptance Criteria",
			},
			wantContains: []string{
				"Automatically modified by Agent",
				"Original content (preserved for reference)",
				"Deploy a Service Mesh on K8s clusters",
				"## Description",
				"## Acceptance Criteria",
			},
			wantNotContains: []string{
				"<!-- 🤖 Agent Modified --><!-- /Agent Modified -->", // Should not be empty
			},
		},
		{
			name:         "removes existing agent notice",
			originalBody: "<!-- 🤖 Agent Modified -->\nSome content\n<!-- /Agent Modified -->\nOriginal content here",
			fixedBody:    "Fixed content",
			violations:   []string{"Missing required section: Description"},
			wantContains: []string{
				"Fixed content",
				"Original content here",
			},
			wantNotContains: []string{
				"Some content", // Should be removed (it was in the old agent notice)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{}

			got := v.preserveOriginalWithModifications(tt.originalBody, tt.fixedBody, tt.violations)

			// Check that all required strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("preserveOriginalWithModifications() missing required content: %s", want)
					t.Errorf("Got output:\n%s", got)
				}
			}

			// Check that unwanted strings are not present
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(got, notWant) {
					t.Errorf("preserveOriginalWithModifications() contains unwanted content: %s", notWant)
					t.Errorf("Got output:\n%s", got)
				}
			}
		})
	}
}

func TestValidator_RemoveExistingAgentNotice(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		want     string
	}{
		{
			name: "removes agent notice from middle",
			body: "Before\n<!-- 🤖 Agent Modified -->\nNotice content\n<!-- /Agent Modified -->\nAfter",
			want: "Before\n\nAfter",
		},
		{
			name: "removes agent notice from start",
			body: "<!-- 🤖 Agent Modified -->\nNotice\n<!-- /Agent Modified -->\nContent",
			want: "Content",
		},
		{
			name: "removes agent notice from end",
			body: "Content\n<!-- 🤖 Agent Modified -->\nNotice\n<!-- /Agent Modified -->",
			want: "Content",
		},
		{
			name: "no agent notice",
			body: "Just regular content",
			want: "Just regular content",
		},
		{
			name: "malformed notice (no end marker)",
			body: "Content\n<!-- 🤖 Agent Modified -->\nNotice",
			want: "Content\n<!-- 🤖 Agent Modified -->\nNotice", // Should remain unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{}
			got := v.removeExistingAgentNotice(tt.body, "<!-- 🤖 Agent Modified -->", "<!-- /Agent Modified -->")
			if got != tt.want {
				t.Errorf("removeExistingAgentNotice() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateAndFix_Integration(t *testing.T) {
	// Skip integration test that requires LLM client
	// This would require setting up a proper mock or test LLM service
	// In a real scenario, you'd use an interface for the LLM client to enable mocking
	t.Skip("Integration test requires LLM client interface for proper mocking")
}

func TestValidator_ValidateAndFix_ValidIssue(t *testing.T) {
	mockGH := newMockGitHubClient()
	
	rules := TaskFormatRules{
		RequiredSections:     []string{"Description", "Acceptance Criteria"},
		MinDescriptionLength: 50,
		RequireLabels:        true,
		LabelPrefix:          "priority:",
	}

	// For valid issues, we can test without LLM since it's not called
	// Instead, test the checkFormat function directly
	v := &Validator{
		githubClient: mockGH,
		rules:        rules,
	}

	issue := &github.Issue{
		Number: 456,
		Title:  "Service Mesh on K8s Cluster",
		Body:   "Deploy a Service Mesh on K8s clusters with the aim to improve service-to-service communication reliability and observability.\n\n## Description\n\nThis task involves deploying a service mesh solution.\n\n## Acceptance Criteria\n\n- Service mesh deployed\n- Monitoring enabled",
		Labels: []string{"priority:high"},
		URL:    "https://github.com/testorg/testrepo/issues/456",
	}

	// Test checkFormat directly for valid issue
	violations := v.checkFormat(issue)
	if len(violations) > 0 {
		t.Errorf("checkFormat() should return no violations for valid issue, got: %v", violations)
	}
}

