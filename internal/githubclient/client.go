package githubclient

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *github.Client
}

func NewClient(token string) *Client {
	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	return &Client{
		client: github.NewClient(oauthClient),
	}
}

// PaginatedRequest represents a function that makes a paginated API request
type PaginatedRequest[T any] func(ctx context.Context, page int) ([]T, *github.Response, error)

// paginate is a generic function that handles pagination for GitHub API requests
// It takes a PaginatedRequest function and returns all items across all pages.
func paginate[T any](ctx context.Context, request PaginatedRequest[T]) ([]T, error) {
	var allItems []T
	page := 1

	for {
		items, resp, err := request(ctx, page)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	return allItems, nil
}

// ListOrganizationRepositories retrieves all repositories for a given organization.
//
// Parameters:
// - ctx: The context for the API request, used for cancellation and timeouts.
// - org: The name of the organization for which to list repositories.
//
// Returns:
// - A slice of pointers to github.Repository objects representing the repositories.
// - An error if the API request fails.
func (client *Client) ListOrganizationRepositories(ctx context.Context, org string) ([]*github.Repository, error) {
	log.Printf("Fetching repositories for organization: %s", org)

	request := func(ctx context.Context, page int) ([]*github.Repository, *github.Response, error) {
		opts := &github.RepositoryListByOrgOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    page,
			},
		}
		return client.client.Repositories.ListByOrg(ctx, org, opts)
	}

	repos, err := paginate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	return repos, nil
}

// GetRepository retrieves a specific repository by its name in an organization.
//
// Parameters:
// - ctx: The context for the API request, used for cancellation and timeouts.
// - org: The name of the organization that owns the repository.
// - repoName: The name of the repository to retrieve.
//
// Returns:
// - A pointer to a github.Repository object representing the repository.
// - An error if the API request fails or if the repository is not found.
func (client *Client) GetRepository(ctx context.Context, org, repoName string) (*github.Repository, error) {
	log.Printf("Fetching repository: %s/%s", org, repoName)

	repo, _, err := client.client.Repositories.Get(ctx, org, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

// ListRepositoryCollaborators retrieves all collaborators for a given repository in an organization.
//
// Parameters:
// - ctx: The context for the API request, used for cancellation and timeouts.
// - org: The name of the organization that owns the repository.
// - repo: The name of the repository for which to list collaborators.
//
// Returns:
// - A slice of pointers to github.User objects representing the collaborators.
// - An error if the API request fails or if there are issues retrieving the data.
func (client *Client) ListRepositoryCollaborators(ctx context.Context, org, repo string) ([]*github.User, error) {
	request := func(ctx context.Context, page int) ([]*github.User, *github.Response, error) {
		opts := &github.ListCollaboratorsOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    page,
			},
		}
		return client.client.Repositories.ListCollaborators(ctx, org, repo, opts)
	}

	collaborators, err := paginate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to list collaborators: %w", err)
	}

	return collaborators, nil
}

// GetRepositoryPermissionLevel retrieves the permission level of a specific user for a given repository.
//
// Parameters:
// - ctx: The context for the API request.
// - org: The name of the organization that owns the repository.
// - repo: The name of the repository.
// - username: The username of the user whose permission level is being queried.
//
// Returns:
// - A pointer to a github.RepositoryPermissionLevel object containing the user's permission level.
// - An error if the API request fails.
func (client *Client) GetRepositoryPermissionLevel(ctx context.Context, org, repo, username string) (*github.RepositoryPermissionLevel, error) {
	permission, _, err := client.client.Repositories.GetPermissionLevel(ctx, org, repo, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission level: %w", err)
	}
	return permission, nil
}

// ListRepositoryTeams retrieves all the teams that have access to a specific repository.
//
// Parameters:
// - ctx: The context for the API request, used for cancellation and timeouts.
// - org: The name of the organization that owns the repository.
// - repo: The name of the repository for which to list teams.
//
// Returns:
// - A slice of pointers to github.Team objects representing the teams with access to the repository.
// - An error if the API request fails.
func (client *Client) ListRepositoryTeams(ctx context.Context, org, repo string) ([]*github.Team, error) {
	request := func(ctx context.Context, page int) ([]*github.Team, *github.Response, error) {
		opts := &github.ListOptions{
			PerPage: 100,
			Page:    page,
		}
		return client.client.Repositories.ListTeams(ctx, org, repo, opts)
	}

	teams, err := paginate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	return teams, nil
}

// ListRepositoryBranches retrieves a list of branches for the specified repository.
//
// Parameters:
//   - ctx: The context for the API request, used for cancellation and timeouts.
//   - org: The name of the organization that owns the repository.
//   - repo: The name of the repository for which to list branches.
//
// Returns:
//   - A slice of pointers to github.Branch objects representing the branches in the repository.
//   - An error if the operation fails.
func (client *Client) ListRepositoryBranches(ctx context.Context, org, repo string) ([]*github.Branch, error) {
	request := func(ctx context.Context, page int) ([]*github.Branch, *github.Response, error) {
		opts := &github.BranchListOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    page,
			},
		}
		return client.client.Repositories.ListBranches(ctx, org, repo, opts)
	}

	branches, err := paginate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	return branches, nil
}

// GetBranchProtection retrieves the branch protection rules for a specific branch in a repository.
//
// Parameters:
// - ctx: The context for the API request, used for cancellation and timeouts.
// - org: The name of the organization that owns the repository.
// - repo: The name of the repository.
// - branch: The name of the branch for which to retrieve protection rules.
//
// Returns:
// - A pointer to a github.Protection object containing the branch protection rules.
// - An error if the operation fails.
func (client *Client) GetBranchProtection(ctx context.Context, org, repo, branch string) (*github.Protection, error) {
	protection, _, err := client.client.Repositories.GetBranchProtection(ctx, org, repo, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch protection: %w", err)
	}
	return protection, nil
}
