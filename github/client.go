package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *github.Client
	owner  string
	repo   string
}

type Issue struct {
	Number    int
	Title     string
	Body      string
	State     string
	Labels    []string
	Assignee  string
	CreatedAt time.Time
	UpdatedAt time.Time
	URL       string
}

func NewClient(token, owner, repo, baseURL string) (*Client, error) {
	return NewClientWithAuth(token, nil, owner, repo, baseURL)
}

// NewClientWithAuth creates a client with either token or GitHub App authentication
func NewClientWithAuth(token string, appAuth *AppAuth, owner, repo, baseURL string) (*Client, error) {
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

	return &Client{
		client: client,
		owner:  owner,
		repo:   repo,
	}, nil
}

func (c *Client) ListIssues(ctx context.Context, state string) ([]*Issue, error) {
	opts := &github.IssueListByRepoOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allIssues []*github.Issue
	for {
		issues, resp, err := c.client.Issues.ListByRepo(ctx, c.owner, c.repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list issues: %w", err)
		}
		// Filter out pull requests - only include actual issues
		for _, issue := range issues {
			// If PullRequestLinks is not nil, it's a PR, not an issue
			if issue.PullRequestLinks == nil {
				allIssues = append(allIssues, issue)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	result := make([]*Issue, len(allIssues))
	for i, issue := range allIssues {
		labels := make([]string, len(issue.Labels))
		for j, label := range issue.Labels {
			labels[j] = label.GetName()
		}

		assignee := ""
		if issue.Assignee != nil {
			assignee = issue.Assignee.GetLogin()
		}

		result[i] = &Issue{
			Number:    issue.GetNumber(),
			Title:     issue.GetTitle(),
			Body:      issue.GetBody(),
			State:     issue.GetState(),
			Labels:    labels,
			Assignee:  assignee,
			CreatedAt: issue.GetCreatedAt().Time,
			UpdatedAt: issue.GetUpdatedAt().Time,
			URL:       issue.GetHTMLURL(),
		}
	}

	return result, nil
}

// GetIssue gets an issue from the configured repository
func (c *Client) GetIssue(ctx context.Context, number int) (*Issue, error) {
	return c.GetIssueFromRepo(ctx, c.owner, c.repo, number)
}

// GetIssueFromRepo gets an issue from a specific repository
func (c *Client) GetIssueFromRepo(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	issue, _, err := c.client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	// Filter out pull requests - only return actual issues
	if issue.PullRequestLinks != nil {
		return nil, fmt.Errorf("issue #%d is a pull request, not an issue", number)
	}

	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label.GetName()
	}

	assignee := ""
	if issue.Assignee != nil {
		assignee = issue.Assignee.GetLogin()
	}

	return &Issue{
		Number:    issue.GetNumber(),
		Title:     issue.GetTitle(),
		Body:      issue.GetBody(),
		State:     issue.GetState(),
		Labels:    labels,
		Assignee:  assignee,
		CreatedAt: issue.GetCreatedAt().Time,
		UpdatedAt: issue.GetUpdatedAt().Time,
		URL:       issue.GetHTMLURL(),
	}, nil
}

// UpdateIssue updates an issue (implements UnifiedClient interface)
// In repo mode, owner and repo parameters are ignored
func (c *Client) UpdateIssue(ctx context.Context, owner, repo string, number int, title, body *string) error {
	issue := &github.IssueRequest{}
	if title != nil {
		issue.Title = title
	}
	if body != nil {
		issue.Body = body
	}

	_, _, err := c.client.Issues.Edit(ctx, c.owner, c.repo, number, issue)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}
	return nil
}

// AddComment adds a comment to an issue (implements UnifiedClient interface)
// In repo mode, owner and repo parameters are ignored
func (c *Client) AddComment(ctx context.Context, owner, repo string, number int, comment string) error {
	commentReq := &github.IssueComment{
		Body: github.String(comment),
	}

	_, _, err := c.client.Issues.CreateComment(ctx, c.owner, c.repo, number, commentReq)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}
	return nil
}

// GetMode returns the client mode
func (c *Client) GetMode() string {
	return "repo"
}

// CreateIssue creates an issue (implements UnifiedClient interface)
// In repo mode, owner and repo parameters are ignored
func (c *Client) CreateIssue(ctx context.Context, owner, repo, title, body string, labels []string) (*Issue, error) {
	issueReq := &github.IssueRequest{
		Title:  github.String(title),
		Body:   github.String(body),
		Labels: &labels,
	}

	issue, _, err := c.client.Issues.Create(ctx, c.owner, c.repo, issueReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	resultLabels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		resultLabels[i] = label.GetName()
	}

	assignee := ""
	if issue.Assignee != nil {
		assignee = issue.Assignee.GetLogin()
	}

	return &Issue{
		Number:    issue.GetNumber(),
		Title:     issue.GetTitle(),
		Body:      issue.GetBody(),
		State:     issue.GetState(),
		Labels:    resultLabels,
		Assignee:  assignee,
		CreatedAt: issue.GetCreatedAt().Time,
		UpdatedAt: issue.GetUpdatedAt().Time,
		URL:       issue.GetHTMLURL(),
	}, nil
}

// AddLabel adds a label to an issue (implements UnifiedClient interface)
// In repo mode, owner and repo parameters are ignored
func (c *Client) AddLabel(ctx context.Context, owner, repo string, number int, label string) error {
	// Get current issue to retrieve existing labels
	issue, _, err := c.client.Issues.Get(ctx, c.owner, c.repo, number)
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

	_, _, err = c.client.Issues.Edit(ctx, c.owner, c.repo, number, issueReq)
	if err != nil {
		return fmt.Errorf("failed to add label: %w", err)
	}

	return nil
}
