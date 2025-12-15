package github

import (
	"context"
	"fmt"
	"strings"
)

// UnifiedClient provides a unified interface that works with both repo and project modes
type UnifiedClient interface {
	ListIssues(ctx context.Context, state string) ([]*Issue, error)
	GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error)
	UpdateIssue(ctx context.Context, owner, repo string, number int, title, body *string) error
	AddComment(ctx context.Context, owner, repo string, number int, comment string) error
	CreateIssue(ctx context.Context, owner, repo, title, body string, labels []string) (*Issue, error)
	AddLabel(ctx context.Context, owner, repo string, number int, label string) error
	GetMode() string // Returns "repo" or "project"
}

// UnifiedClientWrapper wraps either a Client or ProjectClient to provide unified interface
type UnifiedClientWrapper struct {
	repoClient    *Client
	projectClient *ProjectClient
	mode          string
	repos         []Repository
}

// NewUnifiedClient creates a unified client based on configuration
func NewUnifiedClient(token, owner, repo, projectID string, repos []Repository, baseURL string) (UnifiedClient, error) {
	return NewUnifiedClientWithAuth(token, nil, owner, repo, projectID, repos, baseURL)
}

// NewUnifiedClientWithAuth creates a unified client with either token or GitHub App authentication
func NewUnifiedClientWithAuth(token string, appAuth *AppAuth, owner, repo, projectID string, repos []Repository, baseURL string) (UnifiedClient, error) {
	if projectID != "" {
		// Project mode
		projectClient, err := NewProjectClientWithAuth(token, appAuth, owner, projectID, baseURL)
		if err != nil {
			return nil, err
		}

		return &UnifiedClientWrapper{
			projectClient: projectClient,
			mode:          "project",
			repos:         repos,
		}, nil
	}

	// Repo mode
	repoClient, err := NewClientWithAuth(token, appAuth, owner, repo, baseURL)
	if err != nil {
		return nil, err
	}

	return &UnifiedClientWrapper{
		repoClient: repoClient,
		mode:       "repo",
	}, nil
}

func (uc *UnifiedClientWrapper) GetMode() string {
	return uc.mode
}

func (uc *UnifiedClientWrapper) ListIssues(ctx context.Context, state string) ([]*Issue, error) {
	if uc.mode == "project" {
		// Convert RepositoryConfig to Repository
		repos := make([]Repository, len(uc.repos))
		for i, r := range uc.repos {
			repos[i] = Repository{Owner: r.Owner, Name: r.Name}
		}

		projectIssues, err := uc.projectClient.ListProjectIssues(ctx, state, repos)
		if err != nil {
			return nil, err
		}

		// Convert ProjectIssue to Issue
		issues := make([]*Issue, len(projectIssues))
		for i, pi := range projectIssues {
			issues[i] = &pi.Issue
		}

		return issues, nil
	}

	return uc.repoClient.ListIssues(ctx, state)
}

func (uc *UnifiedClientWrapper) GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	if uc.mode == "project" {
		if owner != "" && repo != "" {
			// Specific repo provided
			projectIssue, err := uc.projectClient.GetProjectIssue(ctx, owner, repo, number)
			if err != nil {
				return nil, err
			}
			return &projectIssue.Issue, nil
		}

		// No repo specified - search across all repos
		return uc.findIssueAcrossRepos(ctx, number)
	}

	// In repo mode, owner and repo are ignored (use client's default)
	return uc.repoClient.GetIssue(ctx, number)
}

// findIssueAcrossRepos searches for an issue across all repositories in project mode
func (uc *UnifiedClientWrapper) findIssueAcrossRepos(ctx context.Context, number int) (*Issue, error) {
	// Try each repository until we find the issue
	for _, repo := range uc.repos {
		projectIssue, err := uc.projectClient.GetProjectIssue(ctx, repo.Owner, repo.Name, number)
		if err == nil {
			// Found it!
			return &projectIssue.Issue, nil
		}
		// Continue searching other repos (ignore errors, just try next repo)
	}

	return nil, fmt.Errorf("issue #%d not found in any repository", number)
}

func (uc *UnifiedClientWrapper) UpdateIssue(ctx context.Context, owner, repo string, number int, title, body *string) error {
	if uc.mode == "project" {
		return uc.projectClient.UpdateProjectIssue(ctx, owner, repo, number, title, body)
	}

	// In repo mode, owner and repo are ignored
	return uc.repoClient.UpdateIssue(ctx, "", "", number, title, body)
}

func (uc *UnifiedClientWrapper) AddComment(ctx context.Context, owner, repo string, number int, comment string) error {
	if uc.mode == "project" {
		return uc.projectClient.AddProjectComment(ctx, owner, repo, number, comment)
	}

	// In repo mode, owner and repo are ignored
	return uc.repoClient.AddComment(ctx, "", "", number, comment)
}

func (uc *UnifiedClientWrapper) CreateIssue(ctx context.Context, owner, repo, title, body string, labels []string) (*Issue, error) {
	if uc.mode == "project" {
		if owner == "" || repo == "" {
			// If no repo specified, use the first repository from the project
			if len(uc.repos) == 0 {
				return nil, fmt.Errorf("no repositories configured for project mode")
			}
			owner = uc.repos[0].Owner
			repo = uc.repos[0].Name
		}

		projectIssue, err := uc.projectClient.CreateProjectIssue(ctx, owner, repo, title, body, labels)
		if err != nil {
			return nil, err
		}
		return &projectIssue.Issue, nil
	}

	// In repo mode, owner and repo are ignored
	return uc.repoClient.CreateIssue(ctx, "", "", title, body, labels)
}

func (uc *UnifiedClientWrapper) AddLabel(ctx context.Context, owner, repo string, number int, label string) error {
	if uc.mode == "project" {
		if owner == "" || repo == "" {
			// If no repo specified, try to find the issue first
			issue, err := uc.GetIssue(ctx, "", "", number)
			if err != nil {
				return fmt.Errorf("failed to find issue: %w", err)
			}
			// Extract owner/repo from issue URL
			owner, repo = extractRepoFromURL(issue.URL)
			if owner == "" || repo == "" {
				return fmt.Errorf("could not determine repository for issue #%d", number)
			}
		}
		return uc.projectClient.AddLabel(ctx, owner, repo, number, label)
	}

	// In repo mode, owner and repo are ignored
	return uc.repoClient.AddLabel(ctx, "", "", number, label)
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
