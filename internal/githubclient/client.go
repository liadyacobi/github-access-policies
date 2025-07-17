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

func (client *Client) ListOrganizationRepositories(ctx context.Context, org string) ([]*github.Repository, error) {
	log.Printf("Fetching repositories for organization: %s", org)

	reqOpts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := client.client.Repositories.ListByOrg(ctx, org, reqOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		reqOpts.Page = resp.NextPage
	}

	return allRepos, nil
}

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
	opts := &github.ListCollaboratorsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allCollaborators []*github.User
	for {
		collaborators, resp, err := client.client.Repositories.ListCollaborators(ctx, org, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list collaborators: %w", err)
		}

		allCollaborators = append(allCollaborators, collaborators...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allCollaborators, nil
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

func (client *Client) ListRepositoryTeams(ctx context.Context, org, repo string) ([]*github.Team, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	var allTeams []*github.Team
	for {
		teams, resp, err := client.client.Repositories.ListTeams(ctx, org, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list teams: %w", err)
		}

		allTeams = append(allTeams, teams...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allTeams, nil
}

func (client *Client) ListRepositoryBranches(ctx context.Context, org, repo string) ([]*github.Branch, error) {
	branches, _, err := client.client.Repositories.ListBranches(ctx, org, repo, &github.BranchListOptions{})
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
