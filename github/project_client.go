package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// ProjectClient handles GitHub Projects (v2) which can span multiple repositories
type ProjectClient struct {
	client    *github.Client
	projectID string // Project number (as string) or GraphQL node ID
	owner     string // Organization or user that owns the project
}

// ProjectIssue represents an issue from a GitHub Project (may be from any linked repo)
type ProjectIssue struct {
	Issue
	RepositoryOwner string
	RepositoryName  string
	RepositoryURL   string
	ProjectItemID   string // GraphQL node ID of the project item
}

// NewProjectClient creates a client for GitHub Projects
func NewProjectClient(token, owner, projectID, baseURL string) (*ProjectClient, error) {
	return NewProjectClientWithAuth(token, nil, owner, projectID, baseURL)
}

// NewProjectClientWithAuth creates a project client with either token or GitHub App authentication
func NewProjectClientWithAuth(token string, appAuth *AppAuth, owner, projectID, baseURL string) (*ProjectClient, error) {
	ctx := context.Background()
	var client *github.Client

	if appAuth != nil {
		// Use GitHub App authentication
		ghClient, err := CreateGitHubClientWithApp(ctx, appAuth)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub client with app auth: %w", err)
		}
		client = ghClient
	} else if token != "" {
		// Use token authentication (legacy)
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)

		if baseURL != "" && baseURL != "https://api.github.com" {
			var err error
			client, err = github.NewClient(tc).WithEnterpriseURLs(baseURL, baseURL)
			if err != nil {
				return nil, fmt.Errorf("failed to create GitHub Enterprise client: %w", err)
			}
		} else {
			client = github.NewClient(tc)
		}
	} else {
		return nil, fmt.Errorf("either token or GitHub App credentials must be provided")
	}

	return &ProjectClient{
		client:    client,
		projectID: projectID,
		owner:     owner,
	}, nil
}

// ListProjectIssues lists all issues in a GitHub Project across all linked repositories
// Note: GitHub Projects v2 uses GraphQL API, but we'll use REST API workaround
// by querying issues from all repositories that might be linked to the project
func (pc *ProjectClient) ListProjectIssues(ctx context.Context, state string, repos []Repository) ([]*ProjectIssue, error) {
	var allIssues []*ProjectIssue

	// Query issues from each repository in the project
	for _, repo := range repos {
		opts := &github.IssueListByRepoOptions{
			State: state,
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}

		for {
			issues, resp, err := pc.client.Issues.ListByRepo(ctx, repo.Owner, repo.Name, opts)
			if err != nil {
				// Log error but continue with other repos
				fmt.Printf("Warning: failed to list issues from %s/%s: %v\n", repo.Owner, repo.Name, err)
				break
			}

			for _, issue := range issues {
				labels := make([]string, len(issue.Labels))
				for j, label := range issue.Labels {
					labels[j] = label.GetName()
				}

				assignee := ""
				if issue.Assignee != nil {
					assignee = issue.Assignee.GetLogin()
				}

				projectIssue := &ProjectIssue{
					Issue: Issue{
						Number:    issue.GetNumber(),
						Title:     issue.GetTitle(),
						Body:      issue.GetBody(),
						State:     issue.GetState(),
						Labels:    labels,
						Assignee:  assignee,
						CreatedAt: issue.GetCreatedAt().Time,
						UpdatedAt: issue.GetUpdatedAt().Time,
						URL:       issue.GetHTMLURL(),
					},
					RepositoryOwner: repo.Owner,
					RepositoryName:  repo.Name,
					RepositoryURL:   fmt.Sprintf("https://github.com/%s/%s", repo.Owner, repo.Name),
				}

				allIssues = append(allIssues, projectIssue)
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}

	return allIssues, nil
}

// GetProjectIssue gets a specific issue from a repository
func (pc *ProjectClient) GetProjectIssue(ctx context.Context, owner, repo string, number int) (*ProjectIssue, error) {
	issue, _, err := pc.client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label.GetName()
	}

	assignee := ""
	if issue.Assignee != nil {
		assignee = issue.Assignee.GetLogin()
	}

	return &ProjectIssue{
		Issue: Issue{
			Number:    issue.GetNumber(),
			Title:     issue.GetTitle(),
			Body:      issue.GetBody(),
			State:     issue.GetState(),
			Labels:    labels,
			Assignee:  assignee,
			CreatedAt: issue.GetCreatedAt().Time,
			UpdatedAt: issue.GetUpdatedAt().Time,
			URL:       issue.GetHTMLURL(),
		},
		RepositoryOwner: owner,
		RepositoryName:  repo,
		RepositoryURL:   fmt.Sprintf("https://github.com/%s/%s", owner, repo),
	}, nil
}

// UpdateProjectIssue updates an issue in a specific repository
func (pc *ProjectClient) UpdateProjectIssue(ctx context.Context, owner, repo string, number int, title, body *string) error {
	issue := &github.IssueRequest{}
	if title != nil {
		issue.Title = title
	}
	if body != nil {
		issue.Body = body
	}

	_, _, err := pc.client.Issues.Edit(ctx, owner, repo, number, issue)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}
	return nil
}

// AddProjectComment adds a comment to an issue in a specific repository
func (pc *ProjectClient) AddProjectComment(ctx context.Context, owner, repo string, number int, comment string) error {
	commentReq := &github.IssueComment{
		Body: github.String(comment),
	}

	_, _, err := pc.client.Issues.CreateComment(ctx, owner, repo, number, commentReq)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}
	return nil
}

// CreateProjectIssue creates an issue in a specific repository
func (pc *ProjectClient) CreateProjectIssue(ctx context.Context, owner, repo, title, body string, labels []string) (*ProjectIssue, error) {
	issueReq := &github.IssueRequest{
		Title:  github.String(title),
		Body:   github.String(body),
		Labels: &labels,
	}

	issue, _, err := pc.client.Issues.Create(ctx, owner, repo, issueReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	resultLabels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		resultLabels[i] = label.GetName()
	}

	resultAssignee := ""
	if issue.Assignee != nil {
		resultAssignee = issue.Assignee.GetLogin()
	}

	return &ProjectIssue{
		Issue: Issue{
			Number:    issue.GetNumber(),
			Title:     issue.GetTitle(),
			Body:      issue.GetBody(),
			State:     issue.GetState(),
			Labels:    resultLabels,
			Assignee:  resultAssignee,
			CreatedAt: issue.GetCreatedAt().Time,
			UpdatedAt: issue.GetUpdatedAt().Time,
			URL:       issue.GetHTMLURL(),
		},
		RepositoryOwner: owner,
		RepositoryName:  repo,
		RepositoryURL:   fmt.Sprintf("https://github.com/%s/%s", owner, repo),
	}, nil
}

// AddLabel adds a label to an issue in a specific repository
func (pc *ProjectClient) AddLabel(ctx context.Context, owner, repo string, number int, label string) error {
	// Get current issue to retrieve existing labels
	issue, _, err := pc.client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	// Check if label already exists
	for _, existingLabel := range issue.Labels {
		if existingLabel.GetName() == label {
			// Label already exists, nothing to do
			return nil
		}
	}

	// Add the new label to the list
	labels := make([]string, len(issue.Labels)+1)
	for i, l := range issue.Labels {
		labels[i] = l.GetName()
	}
	labels[len(issue.Labels)] = label

	// Update issue with new labels
	issueReq := &github.IssueRequest{
		Labels: &labels,
	}

	_, _, err = pc.client.Issues.Edit(ctx, owner, repo, number, issueReq)
	if err != nil {
		return fmt.Errorf("failed to add label: %w", err)
	}

	return nil
}

// Repository represents a repository linked to a GitHub Project
type Repository struct {
	Owner string
	Name  string
}
