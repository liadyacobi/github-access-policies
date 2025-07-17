package server

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v57/github"
	v1 "github.com/liadyacobi/github-access-policies/gen/v1"
	"github.com/liadyacobi/github-access-policies/internal/githubclient"
	"github.com/liadyacobi/github-access-policies/internal/normalizer"
	"github.com/liadyacobi/github-access-policies/internal/policy"
)

type GithubScannerServer struct {
	v1.UnimplementedGithubScannerServer
	githubToken string
	policyDir   string
}

func NewGithubScannerServer(githubToken, policyDir string) *GithubScannerServer {
	return &GithubScannerServer{
		githubToken: githubToken,
		policyDir:   policyDir,
	}
}

type organizationRepositoryData struct {
	collaborators     map[string][]*github.User
	permissions       map[string]map[string]*github.RepositoryPermissionLevel
	teams             map[string][]*github.Team
	branchProtections map[string]map[string]*github.Protection
}

func (s *GithubScannerServer) GetRepositories(ctx context.Context, req *v1.GetRepositoriesRequest) (*v1.GetRepositoriesResponse, error) {
	log.Printf("Starting repository scan for organization: %s", req.OrganizationName)

	client := githubclient.NewClient(s.githubToken)

	repos, err := client.ListOrganizationRepositories(ctx, req.OrganizationName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	log.Printf("Found %d repositories in organization %s", len(repos), req.OrganizationName)

	// 3. Fetch detailed data for each repository
	allRepoData := &organizationRepositoryData{
		collaborators:     make(map[string][]*github.User),
		permissions:       make(map[string]map[string]*github.RepositoryPermissionLevel),
		teams:             make(map[string][]*github.Team),
		branchProtections: make(map[string]map[string]*github.Protection),
	}
	for _, repo := range repos {
		repoName := repo.GetName()
		repoData, err := s.fetchRepositoryData(ctx, client, req.OrganizationName, repoName)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repository data for %s: %w", repoName, err)
		}

		// Merge the data
		allRepoData.collaborators[repoName] = repoData.collaborators
		allRepoData.permissions[repoName] = repoData.permissions
		allRepoData.teams[repoName] = repoData.teams
		allRepoData.branchProtections[repoName] = repoData.branchProtections
	}

	// 4. Normalize the data
	orgData := normalizer.NormalizeOrganizationData(
		req.OrganizationName,
		repos,
		allRepoData.collaborators,
		allRepoData.permissions,
		allRepoData.teams,
		allRepoData.branchProtections,
	)

	// 5. Initialize policy engine and evaluate policies
	engine, err := policy.NewPolicyEngine(s.policyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize policy engine: %w", err)
	}

	violationsMap, err := engine.EvaluateOrganization(ctx, orgData)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policies: %w", err)
	}

	// 6. Build the response
	response := s.buildResponse(repos, violationsMap, orgData)

	log.Printf("Scan completed. Found %d violations across %d repositories", len(violationsMap), len(repos))
	return response, nil
}

// repositoryData holds all the fetched data for a single repository
type repositoryData struct {
	collaborators     []*github.User
	permissions       map[string]*github.RepositoryPermissionLevel
	teams             []*github.Team
	branchProtections map[string]*github.Protection
}

// fetchRepositoryData fetches all necessary data for a single repository
func (s *GithubScannerServer) fetchRepositoryData(ctx context.Context, client *githubclient.Client, orgName string, repoName string) (*repositoryData, error) {
	log.Printf("Fetching data for repository: %s", repoName)

	// Fetch collaborators
	collaborators, err := client.ListRepositoryCollaborators(ctx, orgName, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch collaborators for %s: %w", repoName, err)
	}

	// TODO: implement concurrent rather than sequential fetching
	// Fetch permissions for each collaborator
	permissions := make(map[string]*github.RepositoryPermissionLevel)
	for _, collab := range collaborators {
		perm, err := client.GetRepositoryPermissionLevel(ctx, orgName, repoName, collab.GetLogin())
		if err != nil {
			return nil, fmt.Errorf("failed to fetch permission for %s/%s: %w", repoName, collab.GetLogin(), err)
		}
		permissions[collab.GetLogin()] = perm
	}

	// Fetch teams
	teams, err := client.ListRepositoryTeams(ctx, orgName, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch teams for %s: %w", repoName, err)
	}

	// Fetch branch protections
	branches, err := client.ListRepositoryBranches(ctx, orgName, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch branches for %s: %w", repoName, err)
	}

	branchProtections := make(map[string]*github.Protection)
	for _, branch := range branches {
		protection, err := client.GetBranchProtection(ctx, orgName, repoName, branch.GetName())
		if err != nil {
			// Branch protection might not exist, which is okay
			log.Printf("No branch protection found for %s/%s (this is normal): %v", repoName, branch.GetName(), err)
			continue
		}
		branchProtections[branch.GetName()] = protection
	}

	return &repositoryData{
		collaborators:     collaborators,
		permissions:       permissions,
		teams:             teams,
		branchProtections: branchProtections,
	}, nil
}

// buildResponse constructs the gRPC response from all the collected data
func (s *GithubScannerServer) buildResponse(repos []*github.Repository, violationsMap map[string][]policy.PolicyViolation, orgData normalizer.OrganizationData) *v1.GetRepositoriesResponse {
	var repositories []*v1.RepositoryScanResult

	for _, repo := range repos {
		repoName := repo.GetName()

		repositories = append(repositories, &v1.RepositoryScanResult{
			Id:                    fmt.Sprintf("%d", repo.GetID()),
			FullName:              repo.GetFullName(),
			HtmlUrl:               repo.GetHTMLURL(),
			Collaborators:         s.convertCollaborators(orgData.Access[repoName].Collaborators),
			TeamAccesses:          s.convertTeamAccesses(orgData.Access[repoName].Teams),
			BranchProtectionRules: s.convertBranchProtectionRules(orgData.Protection[repoName]),
			Violations:            s.convertViolations(violationsMap[repoName]),
			IsPublic:              !repo.GetPrivate(),
			IsFork:                repo.GetFork(),
		})
	}

	return &v1.GetRepositoriesResponse{
		Repositories: repositories,
	}
}

// Helper functions for converting data structures

func (s *GithubScannerServer) convertCollaborators(collaborators map[string]normalizer.CollaboratorData) []*v1.Collaborator {
	var result []*v1.Collaborator
	for _, collab := range collaborators {
		result = append(result, &v1.Collaborator{
			GithubId:   fmt.Sprintf("%d", collab.ID),
			Login:      collab.Login,
			Type:       collab.Type,
			Permission: collab.Permission,
		})
	}
	return result
}

func (s *GithubScannerServer) convertTeamAccesses(teams map[string]normalizer.TeamData) []*v1.TeamAccess {
	var result []*v1.TeamAccess
	for _, team := range teams {
		result = append(result, &v1.TeamAccess{
			TeamId:     fmt.Sprintf("%d", team.ID),
			TeamName:   team.Name,
			Permission: team.Permission,
			Slug:       team.Slug,
		})
	}
	return result
}

func (s *GithubScannerServer) convertBranchProtectionRules(protections []normalizer.BranchProtectionData) []*v1.BranchProtectionRule {
	var result []*v1.BranchProtectionRule
	for _, protectionData := range protections {
		result = append(result, &v1.BranchProtectionRule{
			BranchName:                   protectionData.BranchName,
			RequiredApprovingReviewCount: int32(protectionData.RequiredApprovingReviewCount),
			RestrictPushes:               protectionData.RestrictPushes,
			RestrictedPushUsers:          protectionData.RestrictedPushUsers,
			RestrictedPushTeams:          protectionData.RestrictedPushTeams,
		})
	}
	return result
}

func (s *GithubScannerServer) convertViolations(violations []policy.PolicyViolation) []*v1.PolicyViolation {
	var result []*v1.PolicyViolation
	for _, violation := range violations {
		result = append(result, &v1.PolicyViolation{
			PolicyId:    violation.PolicyID,
			Description: violation.Description,
			Severity:    violation.Severity,
			Details:     violation.Details,
		})
	}
	return result
}
